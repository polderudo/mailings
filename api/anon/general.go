package anon

import (
	"app/internals"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	bb "github.com/nakami-lounge-GmbH/tools/build"
)

func Info(c echo.Context) error {
	info, err := bb.GetBuildInfo()
	if err != nil {
		return err
	}

	log.Printf("Info from IP: %s", internals.GetIP(c))

	return c.JSONPretty(http.StatusOK, info, "  ")
}
