package internals

import "github.com/labstack/echo/v4"

func GetIP(c echo.Context) string {
	ip := c.RealIP()
	if ip == "::1" {
		ip = "127.0.0.1"
	}
	return ip
}
