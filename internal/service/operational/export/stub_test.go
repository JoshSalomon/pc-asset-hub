package export_test

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

type stubExporter struct {
	name   string
	desc   string
	params []export.ParameterDef
	validateErr error
	exportOut   *export.ExportOutput
	exportErr   error
}

func (s *stubExporter) Name() string        { return s.name }
func (s *stubExporter) Description() string  { return s.desc }
func (s *stubExporter) ParameterSchema() []export.ParameterDef { return s.params }
func (s *stubExporter) ValidateSchema(params map[string]string, schema export.SchemaInfo) error {
	return s.validateErr
}
func (s *stubExporter) Export(_ context.Context, _ export.ExportInput) (*export.ExportOutput, error) {
	if s.exportOut != nil {
		return s.exportOut, s.exportErr
	}
	return &export.ExportOutput{}, s.exportErr
}
