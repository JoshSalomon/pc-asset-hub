package main

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/operator/controllers"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	log := ctrl.Log.WithName("operator")

	namespace := os.Getenv("WATCH_NAMESPACE")
	if namespace == "" {
		namespace = "assethub"
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	})
	if err != nil {
		log.Error(err, "unable to create manager")
		fmt.Fprintf(os.Stderr, "unable to create manager: %v\n", err)
		os.Exit(1)
	}

	reconciler := &controllers.AssetHubReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		log.Error(err, "unable to setup controller")
		fmt.Fprintf(os.Stderr, "unable to setup controller: %v\n", err)
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "manager exited with error")
		fmt.Fprintf(os.Stderr, "manager exited with error: %v\n", err)
		os.Exit(1)
	}
}
