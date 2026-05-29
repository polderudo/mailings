package settingsview

import (
	"app/mq"
	"strings"
)

type SettingsProfileFormData struct {
	UserID     int32
	Email      string
	Salutation string
	Forename   string
	Lastname   string
}

func settingsUserGroupsLabel(groups []string) string {
	if len(groups) == 0 {
		return "Not assigned"
	}

	return strings.Join(groups, ", ")
}

func SettingsProfileFormDataFromUser(user *mq.UserProfile) SettingsProfileFormData {
	if user == nil {
		return SettingsProfileFormData{}
	}

	return SettingsProfileFormData{
		UserID:     user.ID,
		Email:      user.Email,
		Salutation: user.Salutation,
		Forename:   user.Forename,
		Lastname:   user.Lastname,
	}
}
