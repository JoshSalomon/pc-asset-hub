package export

import "sort"

type ExporterRegistry struct {
	exporters map[string]Exporter
}

type ExporterInfo struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	ParameterSchema []ParameterDef `json:"parameter_schema"`
}

func NewExporterRegistry() *ExporterRegistry {
	return &ExporterRegistry{exporters: make(map[string]Exporter)}
}

func (r *ExporterRegistry) Register(e Exporter) {
	r.exporters[e.Name()] = e
}

func (r *ExporterRegistry) Get(name string) (Exporter, bool) {
	e, ok := r.exporters[name]
	return e, ok
}

func (r *ExporterRegistry) List() []ExporterInfo {
	items := make([]ExporterInfo, 0, len(r.exporters))
	for _, e := range r.exporters {
		schema := e.ParameterSchema()
		if schema == nil {
			schema = []ParameterDef{}
		}
		items = append(items, ExporterInfo{
			Name:            e.Name(),
			Description:     e.Description(),
			ParameterSchema: schema,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}
