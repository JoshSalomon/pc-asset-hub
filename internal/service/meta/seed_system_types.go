package meta

import (
	"context"
	"log"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// systemTypes defines the immutable system type definitions seeded on startup.
var systemTypes = []struct {
	Name     string
	BaseType models.BaseType
}{
	{"string", models.BaseTypeString},
	{"integer", models.BaseTypeInteger},
	{"number", models.BaseTypeNumber},
	{"boolean", models.BaseTypeBoolean},
	{"date", models.BaseTypeDate},
	{"url", models.BaseTypeURL},
	{"json", models.BaseTypeJSON},
}

// SeedSystemTypes ensures all system type definitions exist.
// Idempotent — skips types that already exist.
func SeedSystemTypes(ctx context.Context, svc *TypeDefinitionService, tdRepo repository.TypeDefinitionRepository) error {
	for _, st := range systemTypes {
		_, err := tdRepo.GetByName(ctx, st.Name)
		if err == nil {
			continue // already exists
		}
		if !domainerrors.IsNotFound(err) {
			return err
		}
		if _, err := svc.CreateSystemTypeDefinition(ctx, st.Name, st.BaseType); err != nil {
			return err
		}
		log.Printf("seeded system type definition: %s", st.Name)
	}
	return nil
}
