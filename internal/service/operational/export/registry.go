package export

import (
	"sort"
	"sync"
)

type ExporterRegistry struct {
	mu        sync.RWMutex
	exporters map[string]Exporter
	builtIn   map[string]bool
}

type ExporterInfo struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	ParameterSchema []ParameterDef `json:"parameter_schema"`
	Source          string         `json:"source"`
	Health          string         `json:"health"`
}

func NewExporterRegistry() *ExporterRegistry {
	return &ExporterRegistry{
		exporters: make(map[string]Exporter),
		builtIn:   make(map[string]bool),
	}
}

func (r *ExporterRegistry) Register(e Exporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exporters[e.Name()] = e
	r.builtIn[e.Name()] = true
}

func (r *ExporterRegistry) RegisterWebhook(e Exporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exporters[e.Name()] = e
	r.builtIn[e.Name()] = false
}

func (r *ExporterRegistry) Deregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.builtIn[name] {
		return false
	}
	if _, ok := r.exporters[name]; !ok {
		return false
	}
	delete(r.exporters, name)
	delete(r.builtIn, name)
	return true
}

func (r *ExporterRegistry) IsBuiltIn(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.builtIn[name]
}

func (r *ExporterRegistry) Get(name string) (Exporter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.exporters[name]
	return e, ok
}

func (r *ExporterRegistry) List() []ExporterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ExporterInfo, 0, len(r.exporters))
	for name, e := range r.exporters {
		schema := e.ParameterSchema()
		if schema == nil {
			schema = []ParameterDef{}
		}
		source := "built-in"
		health := "n/a"
		if !r.builtIn[name] {
			source = "webhook"
			if we, ok := e.(*WebhookExporter); ok {
				health = we.HealthStatus()
			}
		}
		items = append(items, ExporterInfo{
			Name:            e.Name(),
			Description:     e.Description(),
			ParameterSchema: schema,
			Source:          source,
			Health:          health,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}
