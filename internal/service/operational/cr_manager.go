package operational

import "context"

// CatalogCRSpec contains the data needed to create or update a Catalog CR.
// Name is used as both the K8s resource name and the catalog name in the spec
// (catalog names are DNS-label compatible, so they are valid K8s names).
type CatalogCRSpec struct {
	Name                string // K8s resource name = catalog name
	Namespace           string
	CatalogVersionLabel string
	ValidationStatus    string
	APIEndpoint         string
	SourceDBID          string
	PublishedAt         string
}

// CatalogCRManager defines the interface for managing Catalog CRs in K8s.
type CatalogCRManager interface {
	CreateOrUpdate(ctx context.Context, spec CatalogCRSpec) error
	Delete(ctx context.Context, name, namespace string) error
}
