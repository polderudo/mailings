package mw

import (
	"app/i18n"

	"github.com/labstack/echo/v4"
)

const LangCookieName = "lang"

func ResolveRequestLang(c echo.Context) string {
	if c == nil {
		return i18n.NormalizeLang("")
	}

	cookie, err := c.Cookie(LangCookieName)
	if err == nil && cookie != nil {
		return i18n.NormalizeLang(cookie.Value)
	}

	return i18n.NormalizeLang("")
}
