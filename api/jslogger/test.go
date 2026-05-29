package jslogger

import (
	"app/mw"
	"fmt"
	"net/http"
	"runtime/debug"
)

func Test(c *mw.AnonUserContext) error {
	return c.JSON(http.StatusOK, "{}")
}

func Info(c *mw.AnonUserContext) error {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("debug.ReadBuildInfo() failed")
	}

	return c.JSON(http.StatusOK, info)
}
