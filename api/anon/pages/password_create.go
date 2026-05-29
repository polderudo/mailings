package pages

import (
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	homeview "app/views/pages/main/home"
)

func PasswordCreatePage(c *mw.AnonUserContext) error {
	if c.HasUser {
		isAdmin := c.HasUser && c.IsAdmin()
		if isHXRequest(c) {
			return views.Render(c, homeview.IndexContent(c.UserProfile, false, isAdmin))
		}
		return views.Render(c, homeview.Index(c.UserProfile, c.Lang, false, isAdmin))
	}

	token := c.Param("token")
	if token == "" {
		return views.Render(c, authshared.Login("", c.Lang))
	}

	return views.Render(c, authshared.PasswordCreate(token, "", c.Lang))
}

