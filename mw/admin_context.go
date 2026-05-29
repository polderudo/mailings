package mw

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func SetAdminContext(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := NewUserContext(c)
		if cc != nil {
			if cc.IsAdmin() {
				return f(cc)
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "no admin")
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "error setting context, no admin")
	}
}
