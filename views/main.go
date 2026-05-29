package views

import (
	"app/internals"
	"bytes"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func Render(c echo.Context, cmp templ.Component) error {
	var buf bytes.Buffer
	err := cmp.Render(c.Request().Context(), &buf)
	if err != nil && internals.IsTemplWatchedStringsError(err) {
		internals.DisableTemplDevMode()
		buf.Reset()
		err = cmp.Render(c.Request().Context(), &buf)
	}
	if err != nil {
		return err
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	_, err = c.Response().Writer.Write(buf.Bytes())
	return err
}
