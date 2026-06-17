package k8s

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

type ExporterPluginWatcher struct {
	registry    *export.ExporterRegistry
	namespace   string
	cache       ctrlcache.Cache
	tokenGetter func() string
}

func NewExporterPluginWatcher(cfg *rest.Config, registry *export.ExporterRegistry, namespace string, tokenGetter func() string) (*ExporterPluginWatcher, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rest.Config is nil: K8s client not available")
	}

	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add scheme: %w", err)
	}

	c, err := ctrlcache.New(cfg, ctrlcache.Options{
		Scheme: scheme,
		DefaultNamespaces: map[string]ctrlcache.Config{
			namespace: {},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &ExporterPluginWatcher{
		registry:    registry,
		namespace:   namespace,
		cache:       c,
		tokenGetter: tokenGetter,
	}, nil
}

func (w *ExporterPluginWatcher) Start(ctx context.Context) error {
	informer, err := w.cache.GetInformer(ctx, &v1alpha1.ExporterPlugin{})
	if err != nil {
		return fmt.Errorf("failed to get informer: %w", err)
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    w.handleAdd,
		UpdateFunc: w.handleUpdate,
		DeleteFunc: w.handleDelete,
	})

	return w.cache.Start(ctx)
}

func (w *ExporterPluginWatcher) handleAdd(obj interface{}) {
	cr, ok := obj.(*v1alpha1.ExporterPlugin)
	if !ok {
		return
	}

	if w.registry.IsBuiltIn(cr.Name) {
		log.Printf("ExporterPlugin CR '%s' skipped: name reserved by built-in exporter", cr.Name)
		return
	}

	we, err := w.buildWebhookExporter(cr)
	if err != nil {
		log.Printf("ExporterPlugin CR '%s' skipped: %v", cr.Name, err)
		return
	}

	w.registry.RegisterWebhook(we)
}

func (w *ExporterPluginWatcher) handleUpdate(oldObj, newObj interface{}) {
	cr, ok := newObj.(*v1alpha1.ExporterPlugin)
	if !ok {
		return
	}

	if w.registry.IsBuiltIn(cr.Name) {
		return
	}

	we, err := w.buildWebhookExporter(cr)
	if err != nil {
		log.Printf("ExporterPlugin CR '%s' update skipped (keeping old): %v", cr.Name, err)
		return
	}

	w.registry.RegisterWebhook(we)
}

func (w *ExporterPluginWatcher) handleDelete(obj interface{}) {
	cr, ok := obj.(*v1alpha1.ExporterPlugin)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		cr, ok = tombstone.Obj.(*v1alpha1.ExporterPlugin)
		if !ok {
			return
		}
	}

	w.registry.Deregister(cr.Name)
}

func (w *ExporterPluginWatcher) buildWebhookExporter(cr *v1alpha1.ExporterPlugin) (*export.WebhookExporter, error) {
	timeoutSeconds := cr.Spec.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 10
	}

	paramDefs := make([]export.ParameterDef, len(cr.Spec.ParameterSchema))
	for i, p := range cr.Spec.ParameterSchema {
		pd := export.ParameterDef{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Required:    p.Required,
		}
		if len(p.AttributeMappings) > 0 {
			pd.AttributeMappings = make([]export.AttributeMapping, len(p.AttributeMappings))
			for j, am := range p.AttributeMappings {
				pd.AttributeMappings[j] = export.AttributeMapping{
					Name:        am.Name,
					Description: am.Description,
					Required:    am.Required,
					Default:     am.Default,
				}
			}
		}
		paramDefs[i] = pd
	}

	we, err := export.NewWebhookExporter(
		cr.Name,
		cr.Spec.Description,
		cr.Spec.Endpoint,
		paramDefs,
		timeoutSeconds,
		w.tokenGetter,
	)
	if err != nil {
		return nil, err
	}

	if cr.Status.Phase != "" {
		we.SetHealthStatus(cr.Status.Phase)
	}

	return we, nil
}


// HandleAddForTest exposes handleAdd for unit testing.
func (w *ExporterPluginWatcher) HandleAddForTest(obj interface{}) {
	w.handleAdd(obj)
}

// HandleUpdateForTest exposes handleUpdate for unit testing.
func (w *ExporterPluginWatcher) HandleUpdateForTest(oldObj, newObj interface{}) {
	w.handleUpdate(oldObj, newObj)
}

// HandleDeleteForTest exposes handleDelete for unit testing.
func (w *ExporterPluginWatcher) HandleDeleteForTest(obj interface{}) {
	w.handleDelete(obj)
}

// NewExporterPluginWatcherForTest creates a watcher without a cache for unit testing.
func NewExporterPluginWatcherForTest(registry *export.ExporterRegistry, tokenGetter func() string) *ExporterPluginWatcher {
	return &ExporterPluginWatcher{
		registry:    registry,
		tokenGetter: tokenGetter,
	}
}

// Registry returns the registry for testing.
func (w *ExporterPluginWatcher) Registry() *export.ExporterRegistry {
	return w.registry
}

var saTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func ServiceAccountTokenGetter() func() string {
	var warnOnce sync.Once
	return func() string {
		token, err := os.ReadFile(saTokenPath)
		if err != nil {
			warnOnce.Do(func() {
				log.Printf("warning: cannot read SA token from %s: %v (webhook auth will be unauthenticated)", saTokenPath, err)
			})
			return ""
		}
		return strings.TrimSpace(string(token))
	}
}

var _ client.Object = &v1alpha1.ExporterPlugin{}
