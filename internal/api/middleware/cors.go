package middleware

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// CORSConfig creates an Echo CORS middleware with the specified allowed origins.
// If origins is nil or empty, CORS is not applied (returns a no-op middleware).
func CORSConfig(origins []string) echo.MiddlewareFunc {
	if len(origins) == 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization", "X-User-Role"},
	})
}
