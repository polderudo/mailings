package auth

import (
	"app/i18n"
	"app/mw"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func SetLanguage(c echo.Context) error {
	lang := strings.TrimSpace(c.Param("lang"))
	if !i18n.IsSupportedLang(lang) {
		return echo.NewHTTPError(http.StatusBadRequest, "unsupported language")
	}

	c.SetCookie(&http.Cookie{
		Name:     mw.LangCookieName,
		Value:    i18n.NormalizeLang(lang),
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   shouldUseSecureCookies(c),
		SameSite: http.SameSiteLaxMode,
	})

	referer := strings.TrimSpace(c.Request().Referer())
	if referer == "" {
		referer = "/"
	}

	return c.Redirect(http.StatusSeeOther, referer)
}
