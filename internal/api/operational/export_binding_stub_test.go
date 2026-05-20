package operational_test

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

type testExporter struct {
	name        string
	desc        string
	params      []export.ParameterDef
	validateErr error
	exportOut   *export.ExportOutput
	exportErr   error
}

func (e *testExporter) Name() string                          { return e.name }
func (e *testExporter) Description() string                   { return e.desc }
func (e *testExporter) ParameterSchema() []export.ParameterDef { return e.params }
func (e *testExporter) ValidateSchema(_ map[string]string, _ export.SchemaInfo) error {
	return e.validateErr
}
func (e *testExporter) Export(_ context.Context, _ export.ExportInput) (*export.ExportOutput, error) {
	if e.exportOut != nil {
		return e.exportOut, e.exportErr
	}
	return &export.ExportOutput{}, e.exportErr
}
