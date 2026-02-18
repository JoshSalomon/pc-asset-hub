package health

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Handler provides health check endpoints.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new health handler.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// Healthz always returns 200 — indicates the process is running.
func (h *Handler) Healthz(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz returns 200 if the database is reachable, 503 otherwise.
func (h *Handler) Readyz(c echo.Context) error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "error", "detail": "cannot get database handle"})
	}
	if err := sqlDB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "error", "detail": "database unreachable"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

// RegisterRoutes registers health check routes on the given Echo instance.
func RegisterRoutes(e *echo.Echo, h *Handler) {
	e.GET("/healthz", h.Healthz)
	e.GET("/readyz", h.Readyz)
}
