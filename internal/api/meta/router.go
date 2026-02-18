package meta

import (
	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
)

// SetupRoutes configures all meta API routes on the given Echo group.
// The group should be /api/meta/v1.
func SetupRoutes(g *echo.Group, etHandler *EntityTypeHandler) {
	// Apply RBAC middleware to all meta routes
	rbacProvider := &middleware.HeaderRBACProvider{}
	g.Use(middleware.RBACMiddleware(rbacProvider))

	requireAdmin := middleware.RequireRole(middleware.RoleAdmin)
	RegisterEntityTypeRoutes(g, etHandler, requireAdmin)
}
