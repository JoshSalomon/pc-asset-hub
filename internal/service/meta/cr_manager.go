package meta

import (
	"context"
	"regexp"
	"strings"
)

// CatalogVersionCRSpec contains the data needed to create or update a CatalogVersion CR.
type CatalogVersionCRSpec struct {
	Name           string
	Namespace      string
	VersionLabel   string
	Description    string
	LifecycleStage string
	EntityTypes    []string
	SourceDBID     string
	PromotedBy     string
	PromotedAt     string
}

// CatalogVersionCRManager defines the interface for managing CatalogVersion CRs in K8s.
type CatalogVersionCRManager interface {
	CreateOrUpdate(ctx context.Context, spec CatalogVersionCRSpec) error
	Delete(ctx context.Context, name, namespace string) error
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// maxK8sNameLength is the maximum length for a K8s resource name (DNS subdomain).
const maxK8sNameLength = 253

// SanitizeK8sName converts a version label into a valid K8s resource name.
// Lowercase, replace non-alphanumeric characters with hyphens, trim leading/trailing hyphens,
// and truncate to 253 characters (K8s DNS subdomain limit).
// Returns an empty string if the input contains no alphanumeric characters.
func SanitizeK8sName(label string) string {
	name := strings.ToLower(label)
	name = nonAlphanumeric.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > maxK8sNameLength {
		name = strings.TrimRight(name[:maxK8sNameLength], "-")
	}
	return name
}
