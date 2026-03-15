package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// CatalogAccessChecker determines whether the current user can access a specific catalog.
type CatalogAccessChecker interface {
	CheckAccess(c echo.Context, catalogName, verb string) (bool, error)
}

// HeaderCatalogAccessChecker always allows access — used in dev mode (RBAC_MODE=header).
type HeaderCatalogAccessChecker struct{}

func (h *HeaderCatalogAccessChecker) CheckAccess(_ echo.Context, _ string, _ string) (bool, error) {
	return true, nil
}

// httpMethodToVerb maps an HTTP method to a K8s-style RBAC verb.
func httpMethodToVerb(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead:
		return "get"
	case http.MethodPost:
		return "create"
	case http.MethodDelete:
		return "delete"
	default:
		return "get"
	}
}

// RequireCatalogAccess returns middleware that checks per-catalog access.
// It extracts the catalog name from the :catalog-name URL param and calls
// the CatalogAccessChecker with the appropriate verb.
// If no catalog name is in the path (e.g., catalog list endpoint), it passes through.
func RequireCatalogAccess(checker CatalogAccessChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			catalogName := c.Param("catalog-name")
			if catalogName == "" {
				return next(c)
			}

			verb := httpMethodToVerb(c.Request().Method)
			allowed, err := checker.CheckAccess(c, catalogName, verb)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "access check failed")
			}
			if !allowed {
				return echo.NewHTTPError(http.StatusForbidden, "access denied to catalog: "+catalogName)
			}

			return next(c)
		}
	}
}

// FilterAccessible filters a slice by checking catalog access for each item.
// The nameFunc extracts the catalog name from each item.
func FilterAccessible[T any](c echo.Context, checker CatalogAccessChecker, items []T, nameFunc func(T) string) ([]T, error) {
	var accessible []T
	for _, item := range items {
		allowed, err := checker.CheckAccess(c, nameFunc(item), "get")
		if err != nil {
			return nil, err
		}
		if allowed {
			accessible = append(accessible, item)
		}
	}
	return accessible, nil
}
