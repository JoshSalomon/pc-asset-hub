package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Role represents a user role.
type Role string

const (
	RoleRO         Role = "RO"
	RoleRW         Role = "RW"
	RoleAdmin      Role = "Admin"
	RoleSuperAdmin Role = "SuperAdmin"
)

// RoleContextKey is the key used to store the role in the Echo context.
const RoleContextKey = "user_role"

// RBACProvider determines the user's role from the request.
type RBACProvider interface {
	GetRole(c echo.Context) (Role, error)
}

// HeaderRBACProvider reads the role from the X-User-Role header (for development/testing).
type HeaderRBACProvider struct{}

func (p *HeaderRBACProvider) GetRole(c echo.Context) (Role, error) {
	roleStr := c.Request().Header.Get("X-User-Role")
	switch Role(roleStr) {
	case RoleRO, RoleRW, RoleAdmin, RoleSuperAdmin:
		return Role(roleStr), nil
	case "":
		return "", echo.NewHTTPError(http.StatusUnauthorized, "missing X-User-Role header")
	default:
		return "", echo.NewHTTPError(http.StatusUnauthorized, "invalid role: "+roleStr)
	}
}

// RBACMiddleware extracts the user role and stores it in the context.
func RBACMiddleware(provider RBACProvider) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, err := provider.GetRole(c)
			if err != nil {
				return err
			}
			c.Set(RoleContextKey, role)
			return next(c)
		}
	}
}

// RequireRole returns middleware that checks if the user has at least the required role.
func RequireRole(minRole Role) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get(RoleContextKey).(Role)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "no role in context")
			}
			if !hasMinRole(role, minRole) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}

// GetRoleFromContext extracts the role from the Echo context.
func GetRoleFromContext(c echo.Context) Role {
	role, _ := c.Get(RoleContextKey).(Role)
	return role
}

func hasMinRole(actual, required Role) bool {
	roleLevel := map[Role]int{
		RoleRO:         0,
		RoleRW:         1,
		RoleAdmin:      2,
		RoleSuperAdmin: 3,
	}
	return roleLevel[actual] >= roleLevel[required]
}
