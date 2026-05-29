package pages

import (
	"app/api/router"
	"app/auth"
	coreauth "app/core/auth"
	"app/db"
	"app/i18n"
	"app/mq"
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	usersview "app/views/pages/main/users"
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
)

func UserDetailPage(c *mw.AnonUserContext) error {
	if !c.HasUser {
		return views.Render(c, authshared.Login("", c.Lang))
	}

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	ctx := context.Background()
	targetUser := auth.GetUserDetails(ctx, db.DBob, int32(userID), true, true)
	if targetUser == nil {
		return fmt.Errorf("user not found")
	}

	userGroups := auth.GetUserGroups(int32(userID))
	allGroups := auth.GetAllUsedGroups()

	if isHXRequest(c) {
		return views.Render(c, usersview.UserDetailPage(c.UserProfile, c.Lang, targetUser, userGroups, allGroups))
	}
	return views.Render(c, usersview.UserDetail(c.UserProfile, c.Lang, targetUser, userGroups, allGroups))
}

func UserUpdateAccount(c *mw.UserContext) error {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	ctx := context.Background()

	rawUser, err := mq.FindUserProfile(ctx, db.DBob, int32(userID))
	if err != nil || rawUser == nil {
		return fmt.Errorf("user not found")
	}

	disabled := c.FormValue("disabled") == "on"
	mustReset := c.FormValue("must_reset_password") == "on"
	newPassword := c.FormValue("new_password")
	confirmPassword := c.FormValue("confirm_password")

	if newPassword != "" && newPassword != confirmPassword {
		targetUser := auth.GetUserDetails(ctx, db.DBob, int32(userID), false, false)
		return views.Render(c, usersview.UserDetailAccountSecurity(
			c.Lang, targetUser,
			i18n.TrLang(c.Lang, i18n.L.Auth.PasswordsDoNotMatch),
			"",
		))
	}
	if err := rawUser.Update(ctx, db.DBob, &mq.UserProfileSetter{
		Disabled:          omit.From(disabled),
		MustResetPassword: omit.From(mustReset),
		UpdatedAt:         omitnull.From(time.Now()),
	}); err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	if newPassword != "" {
		hashedPw, err := coreauth.NewAuth().CryptPassword(newPassword)
		if err != nil {
			targetUser := auth.GetUserDetails(ctx, db.DBob, int32(userID), false, false)
			return views.Render(c, usersview.UserDetailAccountSecurity(
				c.Lang, targetUser, "Unable to encrypt password.", "",
			))
		}
		if err := rawUser.Update(ctx, db.DBob, &mq.UserProfileSetter{
			Password:  omit.From(hashedPw),
			UpdatedAt: omitnull.From(time.Now()),
		}); err != nil {
			return fmt.Errorf("update password: %w", err)
		}
	}

	targetUser := auth.GetUserDetails(ctx, db.DBob, int32(userID), false, false)
	if targetUser == nil {
		return fmt.Errorf("user not found after update")
	}
	return views.RenderWithToast(c,
		usersview.UserDetailAccountSecurity(c.Lang, targetUser, "", ""),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Users.SavedSuccessfully),
	)
}

func UserUpdateGroups(c *mw.UserContext) error {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	ctx := context.Background()

	currentGroups := auth.GetUserGroups(int32(userID))
	formParams, _ := c.FormParams()
	newGroups := formParams["groups"]

	var errs []error

	for _, g := range currentGroups {
		if !slices.Contains(newGroups, g) {
			if err := auth.RemoveUserFromGroup(int32(userID), g); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for _, g := range newGroups {
		if !slices.Contains(currentGroups, g) {
			if err := auth.AddUserToGroup(int32(userID), g); err != nil {
				errs = append(errs, err)
			}
		}
	}

	targetUser := auth.GetUserDetails(ctx, db.DBob, int32(userID), true, true)
	if targetUser == nil {
		return fmt.Errorf("user not found after update")
	}

	updatedGroups := auth.GetUserGroups(int32(userID))
	allGroups := auth.GetAllUsedGroups()

	if len(errs) > 0 {
		return views.RenderWithToast(c,
			usersview.UserDetailGroups(c.Lang, targetUser, updatedGroups, allGroups),
			"error",
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Users.SaveFailed),
		)
	}

	return views.RenderWithToast(c,
		usersview.UserDetailGroups(c.Lang, targetUser, updatedGroups, allGroups),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Users.SavedSuccessfully),
	)
}

func UserDelete(c *mw.UserContext) error {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if int32(userID) == c.UserProfile.ID {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Users.CannotDeleteYourOwnAccount),
		)
	}

	ctx := context.Background()

	user, err := mq.FindUserProfile(ctx, db.DBob, int32(userID))
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	if err := user.Delete(ctx, db.DBob); err != nil {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Users.DeleteFailed),
		)
	}

	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Users.List))
	return c.NoContent(200)
}
