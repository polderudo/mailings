package pages

import (
	"app/auth"
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	settingsview "app/views/pages/main/settings"
)

func SettingsPage(c *mw.AnonUserContext) error {
	if c.HasUser {
		userGroups := auth.GetUserGroups(c.UserProfile.ID)
		if isHXRequest(c) {
			return views.Render(c, settingsview.SettingsPage(c.UserProfile, c.Lang, userGroups))
		}
		return views.Render(c, settingsview.Settings(c.UserProfile, c.Lang, userGroups))
	}
	return views.Render(c, authshared.Login("", c.Lang))
}