package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	apihealth "github.com/project-catalyst/pc-asset-hub/internal/api/health"
	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	"github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	apiop "github.com/project-catalyst/pc-asset-hub/internal/api/operational"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/config"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/database"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
	gormrepo "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	k8sinfra "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/k8s"
	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func main() {
	cfg := config.Load()

	// Database
	db, err := database.NewDB(cfg.DBConnectionString)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := gormmodels.InitDB(db); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// Repositories
	etRepo := gormrepo.NewEntityTypeGormRepo(db)
	etvRepo := gormrepo.NewEntityTypeVersionGormRepo(db)
	attrRepo := gormrepo.NewAttributeGormRepo(db)
	assocRepo := gormrepo.NewAssociationGormRepo(db)
	enumRepo := gormrepo.NewEnumGormRepo(db)
	cvRepo := gormrepo.NewCatalogVersionGormRepo(db)
	pinRepo := gormrepo.NewCatalogVersionPinGormRepo(db)
	ltRepo := gormrepo.NewLifecycleTransitionGormRepo(db)
	instRepo := gormrepo.NewEntityInstanceGormRepo(db)
	iavRepo := gormrepo.NewInstanceAttributeValueGormRepo(db)
	linkRepo := gormrepo.NewAssociationLinkGormRepo(db)

	// Optional K8s client for CatalogVersion CR management
	var crManager svcmeta.CatalogVersionCRManager
	k8sRestConfig, k8sErr := rest.InClusterConfig()
	if k8sErr == nil {
		scheme := runtime.NewScheme()
		_ = v1alpha1.AddToScheme(scheme)
		k8sClient, clientErr := ctrlclient.New(k8sRestConfig, ctrlclient.Options{Scheme: scheme})
		if clientErr != nil {
			log.Printf("warning: failed to create K8s client: %v (CR management disabled)", clientErr)
		} else {
			crManager = k8sinfra.NewK8sCRManager(k8sClient)
			log.Println("K8s client initialized, CatalogVersion CR management enabled")
		}
	} else {
		log.Printf("not running in K8s cluster (CR management disabled): %v", k8sErr)
	}

	watchNamespace := os.Getenv("WATCH_NAMESPACE")
	if watchNamespace == "" {
		watchNamespace = "assethub"
	}

	// Services
	etSvc := svcmeta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)
	svcmeta.WithCatalogRepos(etSvc, pinRepo, cvRepo)
	svcmeta.WithEnumRepo(etSvc, enumRepo)
	attrSvc := svcmeta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, enumRepo)
	enumSvc := svcmeta.NewEnumService(enumRepo, gormrepo.NewEnumValueGormRepo(db), attrRepo)
	assocSvc := svcmeta.NewAssociationService(assocRepo, etvRepo, attrRepo)
	vhSvc := svcmeta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)
	cvSvc := svcmeta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crManager, watchNamespace, cfg.AllowedStages(), etRepo, etvRepo)
	catalogRepo := gormrepo.NewCatalogGormRepo(db)
	enumValRepo := gormrepo.NewEnumValueGormRepo(db)
	instSvc := svcop.NewEntityInstanceService(instRepo, iavRepo, attrRepo, cvRepo, linkRepo)
	catalogSvc := svcop.NewCatalogService(catalogRepo, cvRepo, instRepo)
	instanceSvc := svcop.NewInstanceService(instRepo, iavRepo, catalogRepo, cvRepo, pinRepo, attrRepo, etvRepo, etRepo, enumValRepo, assocRepo, linkRepo)

	// Handlers
	etHandler := apimeta.NewEntityTypeHandler(etSvc)
	attrHandler := apimeta.NewAttributeHandler(attrSvc)
	assocHandler := apimeta.NewAssociationHandler(assocSvc)
	enumHandler := apimeta.NewEnumHandler(enumSvc)
	vhHandler := apimeta.NewVersionHistoryHandler(vhSvc)
	cvHandler := apimeta.NewCatalogVersionHandler(cvSvc)
	opHandler := apiop.NewHandler(instSvc)
	catalogHandler := apiop.NewCatalogHandler(catalogSvc)
	instanceHandler := apiop.NewInstanceHandler(instanceSvc)
	healthHandler := apihealth.NewHandler(db)

	// Echo server
	e := echo.New()
	e.HideBanner = true

	// CORS
	e.Use(middleware.CORSConfig(cfg.CORSAllowedOrigins))

	// Health
	apihealth.RegisterRoutes(e, healthHandler)

	// Meta API
	metaGroup := e.Group("/api/meta/v1")
	rbacProvider := &middleware.HeaderRBACProvider{}
	if cfg.RBACMode == "header" {
		log.Println("WARNING: RBAC is using header-based mode (X-User-Role). This is an insecure development configuration. Do not use in production.")
	}
	metaGroup.Use(middleware.RBACMiddleware(rbacProvider))
	requireAdmin := middleware.RequireRole(middleware.RoleAdmin)
	requireRW := middleware.RequireRole(middleware.RoleRW)
	apimeta.RegisterEntityTypeRoutes(metaGroup, etHandler, requireAdmin)
	apimeta.RegisterAttributeRoutes(metaGroup, attrHandler, requireAdmin)
	apimeta.RegisterAssociationRoutes(metaGroup, assocHandler, requireAdmin)
	apimeta.RegisterEnumRoutes(metaGroup, enumHandler, requireAdmin)
	apimeta.RegisterVersionHistoryRoutes(metaGroup, vhHandler)
	apimeta.RegisterCatalogVersionRoutes(metaGroup, cvHandler, requireRW)

	// Operational API — Catalog CRUD
	catalogGroup := e.Group("/api/data/v1/catalogs")
	catalogGroup.Use(middleware.RBACMiddleware(rbacProvider))
	apiop.RegisterCatalogRoutes(catalogGroup, catalogHandler, requireRW)

	// Operational API — Instance CRUD (under catalogs)
	apiop.RegisterInstanceRoutes(catalogGroup.Group("/:catalog-name"), instanceHandler, requireRW)

	// Operational API — legacy instance routes (to be replaced)
	opGroup := e.Group("/api/data/v1/:catalog-version")
	opGroup.Use(middleware.RBACMiddleware(rbacProvider))
	apiop.RegisterRoutes(opGroup, opHandler)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		addr := fmt.Sprintf(":%d", cfg.APIPort)
		log.Printf("starting API server on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully...")
	if err := e.Shutdown(context.Background()); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
