package pages

import "app/mw"

func isHXRequest(c *mw.AnonUserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}