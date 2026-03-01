package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
)

func mapError(err error) *echo.HTTPError {
	if domainerrors.IsNotFound(err) {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if domainerrors.IsConflict(err) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if domainerrors.IsValidation(err) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if domainerrors.IsForbidden(err) {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}
	if domainerrors.IsCycleDetected(err) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}
	if domainerrors.IsReferencedEnum(err) {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
	}
	if domainerrors.IsDeepCopyRequired(err) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	// Do not leak internal error details to clients
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}
