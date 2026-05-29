package pages

import (
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	homeview "app/views/pages/main/home"
)

func Index(c *mw.AnonUserContext) error {
	if c.HasUser {
		isAdmin := c.HasUser && c.IsAdmin()
		if isHXRequest(c) {
			return views.Render(c, homeview.IndexContent(c.UserProfile, false, isAdmin))
		}
		return views.Render(c, homeview.Index(c.UserProfile, c.Lang, false, isAdmin))
	}
	return views.Render(c, authshared.Login("", c.Lang))
}

func LoginPage(c *mw.AnonUserContext) error {
	if c.HasUser {
		isAdmin := c.HasUser && c.IsAdmin()
		if isHXRequest(c) {
			return views.Render(c, homeview.IndexContent(c.UserProfile, true, isAdmin))
		}
		return views.Render(c, homeview.Index(c.UserProfile, c.Lang, true, isAdmin))
	}
	return views.Render(c, authshared.Login("", c.Lang))
}

func ForgotPasswordPage(c *mw.AnonUserContext) error {
	if c.HasUser {
		isAdmin := c.HasUser && c.IsAdmin()
		if isHXRequest(c) {
			return views.Render(c, homeview.IndexContent(c.UserProfile, false, isAdmin))
		}
		return views.Render(c, homeview.Index(c.UserProfile, c.Lang, false, isAdmin))
	}
	return views.Render(c, authshared.ForgotPassword("", "", c.Lang))
}
