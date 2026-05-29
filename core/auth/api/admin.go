package api

import (
	"app/api/api_error"
	"app/api/validate"
	"app/auth"
	auth2 "app/core/auth"
	"app/db"
	"app/mq"
	"app/mw"
	"app/views"
	settingsview "app/views/pages/main/settings"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "app/i18n"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/go-playground/validator/v10"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type userUpdateRequest struct {
	Salutation        string `json:"salutation" form:"salutation" validate:"required"`
	UserID            int32  `json:"user_id" form:"user_id" validate:"required"`
	Forename          string `json:"forename" form:"forename" validate:"required"`
	Lastname          string `json:"lastname" form:"lastname" validate:"required"`
	Email             string `json:"email" form:"email" validate:"required,email"`
	MustResetPassword bool   `json:"must_reset_password" form:"must_reset_password"`
	Disabled          bool   `json:"disabled" form:"disabled"`
	EmailValidated    bool   `json:"email_validated" form:"email_validated"`
}

func renderPasswordChangeState(c *mw.UserContext, errorMessage string, successMessage string) error {
	return views.Render(c, settingsview.ChangePasswordState(c.UserProfile, errorMessage, successMessage, c.Lang))
}

func renderSettingsState(c *mw.UserContext, user *mq.UserProfile, formData settingsview.SettingsProfileFormData, errorMessage string, successMessage string) error {
	if user == nil {
		user = c.UserProfile
	}
	userGroups := auth.GetUserGroups(user.ID)
	if c.Request().Header.Get("HX-Request") == "true" {
		return views.Render(c, settingsview.UserInfoUpdateState(user, c.Lang, userGroups, formData, errorMessage, successMessage))
	}
	return views.Render(c, settingsview.SettingsState(user, c.Lang, userGroups, formData, errorMessage, successMessage))
}

func userUpdateFormData(userID int32, email string, salutation string, forename string, lastname string) settingsview.SettingsProfileFormData {
	return settingsview.SettingsProfileFormData{
		UserID:     userID,
		Email:      email,
		Salutation: salutation,
		Forename:   forename,
		Lastname:   lastname,
	}
}

func userUpdateValidationMessage(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		return validationErrors.Error()
	}
	return err.Error()
}

func (m *api) applyUserUpdate(c *mw.UserContext, l *userUpdateRequest) (*mq.UserProfile, error) {
	if c.UserProfile.ID != l.UserID {
		if !c.IsAdmin() {
			return nil, api_error.NewWithCode(c, http.StatusBadRequest).Msg("Not authorized")
		}
	}

	ctx := context.Background()
	user, err := mq.FindUserProfile(ctx, db.DBob, l.UserID)
	if err != nil {
		return nil, api_error.NewWithError(c, err).Msg("error finding user")
	}

	s := &mq.UserProfileSetter{
		Salutation: omit.From(l.Salutation),
		Forename:   omit.From(l.Forename),
		Lastname:   omit.From(l.Lastname),
		Email:      omit.From(l.Email),
		UpdatedAt:  omitnull.From(time.Now()),
	}

	if c.IsAdmin() {
		s.MustResetPassword = omit.From(l.MustResetPassword)
		s.Disabled = omit.From(l.Disabled)
		s.EmailValidated = omit.From(l.EmailValidated)
	}

	if err := user.Update(ctx, db.DBob, s); err != nil {
		return nil, api_error.NewWithError(c, err).Msg("error updating user")
	}

	return mq.FindUserProfile(ctx, db.DBob, l.UserID)
}

func (m *api) ListGroups(c *mw.UserContext) error {
	groups, err := mq.UserGroups.Query().All(context.Background(), db.DBob)
	if err != nil {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Error getting groups")
	}
	return c.JSON(http.StatusOK, groups)
}

func (m *api) GroupUpsert(c *mw.UserContext) error {
	type request struct {
		GroupID     string `json:"group_id" validate:"required"`
		Description string `json:"description"`
		IsAdmin     bool   `json:"is_admin"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	ctx := context.Background()
	group, err := mq.UserGroups.Query(mq.SelectWhere.UserGroups.GroupName.EQ(l.GroupID)).One(ctx, db.DBob)
	if db.IsErrorAndNotNoRows(err) {
		return api_error.NewWithError(c, err).Msg("error finding group")
	}

	us := &mq.UserGroupSetter{
		GroupName:       omit.From(l.GroupID),
		CanBeDelete:     omit.From(true),
		Description:     omitnull.From(l.Description),
		IsAdmin:         omit.Val[bool]{},
		ChangedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.Val[time.Time]{},
	}

	if group == nil {
		group, err = mq.UserGroups.Insert(us).One(ctx, db.DBob)
		if err != nil {
			return api_error.NewWithError(c, err).Msg("error inserting group")
		}
	} else {
		if err = group.Update(ctx, db.DBob, us); err != nil {
			return api_error.NewWithError(c, err).Msg("error updating group")
		}
	}

	return c.JSON(http.StatusOK, group)
}

func (m *api) GroupDelete(c *mw.UserContext) error {
	type request struct {
		GroupID string `json:"group_id" validate:"required"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	ctx := context.Background()

	group, err := mq.UserGroups.Query(mq.SelectWhere.UserGroups.GroupName.EQ(l.GroupID)).One(ctx, db.DBob)
	if db.IsErrorAndNotNoRows(err) {
		return api_error.NewWithError(c, err).Msg("error finding group")
	}

	if group != nil && !group.CanBeDelete {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Group can not be deleted")
	}

	//check that there are no more users in the group
	users, err := auth.GetAllUsersForGroup(l.GroupID)
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error getting users for group")
	}
	if len(users) > 0 {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Group can not be deleted. There are still users in the group")
	}

	//remove the policies from cabin
	if err = auth.RemoveGroupPolicies(l.GroupID); err != nil {
		return api_error.NewWithError(c, err).Msg("error removing group policies")
	}

	if group != nil {
		if err = group.Delete(ctx, db.DBob); err != nil {
			return api_error.NewWithError(c, err).Msg("error deleting group")
		}
	}

	return c.JSON(http.StatusOK, "{}")
}

func (m *api) GroupDetails(c *mw.UserContext) error {
	type request struct {
		GroupID string `json:"group_id" query:"group_id" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	type result struct {
		Group       *mq.UserGroup           `json:"group"`
		GroupUsers  []*auth.UserDescription `json:"group_users"`
		Permissions []string                `json:"permissions"`
	}

	g, err := mq.UserGroups.Query(mq.SelectWhere.UserGroups.GroupName.EQ(l.GroupID)).One(context.Background(), db.DBob)
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error finding group")
	}
	ret := &result{
		Group:       g,
		Permissions: auth.GetGroupPermissions(l.GroupID),
	}

	users, err := auth.GetAllUsersForGroup(l.GroupID)
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error getting users for group")
	}
	for _, userID := range users {
		ret.GroupUsers = append(ret.GroupUsers, auth.GetUserDescription(context.Background(), db.DBob, userID, nil))
	}

	return c.JSON(http.StatusOK, ret)
}

func (m *api) UserDetails(c *mw.UserContext) error {
	type request struct {
		UserID int32 `json:"user_id" query:"user_id" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if c.UserProfile.ID != l.UserID {
		if !c.IsAdmin() {
			return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Not authorized")
		}
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, l.UserID, true, true))
}

func (m *api) UserUpdate(c *mw.UserContext) error {
	l, err := validate.ValidateData[userUpdateRequest](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if _, err := m.applyUserUpdate(c, l); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, l.UserID, true, true))
}

// if result's @data-lucide-init is '1'
//                 exit
//             end
//             js(result)
//                 if (window.renderLucideIcons) {
//                     window.renderLucideIcons(result)
//                 }
//             end
//             set @data-lucide-init of result to '1'

func (m *api) UserUpdateSettingsState(c *mw.UserContext) error {
	l, err := validate.ValidateData[userUpdateRequest](c)
	if err != nil {
		formData := settingsview.SettingsProfileFormDataFromUser(c.UserProfile)
		if l != nil {
			formData = userUpdateFormData(l.UserID, l.Email, l.Salutation, l.Forename, l.Lastname)
		}
		return renderSettingsState(c, c.UserProfile, formData, userUpdateValidationMessage(err), "")
	}

	updatedUser, err := m.applyUserUpdate(c, l)
	if err != nil {
		return renderSettingsState(c, c.UserProfile, userUpdateFormData(l.UserID, l.Email, l.Salutation, l.Forename, l.Lastname), err.Error(), "")
	}

	return renderSettingsState(c, updatedUser, settingsview.SettingsProfileFormDataFromUser(updatedUser), "", "Profile updated successfully.")
}

func (m *api) AddUserToGroup(c *mw.UserContext) error {
	type request struct {
		UserEmail string `json:"user_email" query:"user_email"`
		UserID    int32  `json:"user_id" query:"user_id"`
		GroupID   string `json:"group_id" query:"group_id" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}
	userID := l.UserID
	if l.UserEmail != "" {
		q := mq.SelectWhere.UserProfiles
		user, err := mq.UserProfiles.Query(q.Email.EQ(l.UserEmail)).One(context.Background(), db.DBob)
		if err != nil {
			return api_error.NewWithError(c, err).Msg("error finding user")
		}
		userID = user.ID
	}

	if l.UserID == c.UserProfile.ID && !c.IsAdmin() {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg(Tr(LK_you_can_not_add_yourself_to_a_group))
	}

	if l.GroupID == auth.GroupSuperAdmin {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg(Tr(LK_super_admin_can_not_be_added))
	}

	if err := auth.AddUserToGroup(userID, l.GroupID); err != nil {
		return api_error.NewWithError(c, err).Msg("error adding user to group")
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, userID, true, true))
}

func (m *api) RemoveUserFromGroup(c *mw.UserContext) error {
	type request struct {
		UserEmail string `json:"user_email" query:"user_email"`
		UserID    int32  `json:"user_id" query:"user_id"`
		GroupID   string `json:"group_id" query:"group_id" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}
	userID := l.UserID
	if l.UserEmail != "" {
		q := mq.SelectWhere.UserProfiles
		user, err := mq.UserProfiles.Query(q.Email.EQ(l.UserEmail)).One(context.Background(), db.DBob)
		if err != nil {
			return api_error.NewWithError(c, err).Msg("error finding user")
		}
		userID = user.ID
	}

	//We need to check if the group is admin. At least one admin must be present
	//also super-admin can not be removed!

	if l.UserID == c.UserProfile.ID {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("You can not remove yourself from a group!")
	}

	if l.GroupID == auth.GroupSuperAdmin {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Super-admin can not be removed!")
	}
	if l.GroupID == auth.GroupAdmin {
		allAdminUserIDs, err := auth.GetAllUsersForGroup(l.GroupID)
		if err != nil {
			return api_error.NewWithError(c, err).Msg("error getting all admin users")
		}
		if len(allAdminUserIDs) <= 1 {
			return api_error.NewWithCode(c, http.StatusBadRequest).Msg("At least one admin must be present!")
		}
	}

	if err := auth.RemoveUserFromGroup(userID, l.GroupID); err != nil {
		return api_error.NewWithError(c, err).Msg("error adding user to group")
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, userID, true, true))
}

func (m *api) AddGroupPermission(c *mw.UserContext) error {
	type request struct {
		GroupID    string `json:"group_id" validate:"required"`
		Permission string `json:"permission" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if err := auth.AddGroupPermission(l.GroupID, l.Permission, auth.ActionUpsert); err != nil {
		return api_error.NewWithError(c, err).Msg("error adding group permission")
	}

	return c.JSON(http.StatusOK, auth.GetGroupPermissions(l.GroupID))
}

func (m *api) RemoveGroupPermission(c *mw.UserContext) error {
	type request struct {
		GroupID    string `json:"group_id" validate:"required"`
		Permission string `json:"permission" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if err := auth.RemoveGroupPermission(l.GroupID, l.Permission, auth.ActionUpsert); err != nil {
		return api_error.NewWithError(c, err).Msg("error removing group permission")
	}

	return c.JSON(http.StatusOK, auth.GetGroupPermissions(l.GroupID))
}

func (m *api) UserList(c *mw.UserContext) error {
	users, err := mq.UserProfiles.Query(
		sm.OrderBy(mq.UserProfiles.Columns.Forename).Asc(),
		sm.OrderBy(mq.UserProfiles.Columns.Lastname).Asc(),
	).All(context.Background(), db.DBob)

	if err != nil {
		return api_error.NewWithError(c, err).Msg("error finding users")
	}

	var ret []*auth.UserDescription
	for i := 0; i < len(users); i++ {
		u := users[i]

		ret = append(ret, auth.GetUserDescription(context.Background(), db.DBob, 0, u))
	}
	return c.JSON(http.StatusOK, ret)
}

func (m *api) SetUserPassword(c *mw.UserContext) error {
	type request struct {
		UserID          int32  `json:"user_id" form:"user_id"`
		Password        string `json:"password" form:"password" validate:"required"`
		ConfirmPassword string `json:"confirm_password" form:"confirm_password" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return renderPasswordChangeState(c, "Please fill out both password fields.", "")
		}
		return validate.ValidationError(c, err)
	}

	if l.UserID == 0 {
		l.UserID = c.UserProfile.ID
	}

	l.Password = strings.TrimSpace(l.Password)
	l.ConfirmPassword = strings.TrimSpace(l.ConfirmPassword)

	if len(l.Password) < 8 {
		if c.Request().Header.Get("HX-Request") == "true" {
			return renderPasswordChangeState(c, "Password must be at least 8 characters long.", "")
		}
		return api_error.NewValidation(c).Msg("Password must be at least 8 characters long.")
	}

	if l.Password != l.ConfirmPassword {
		if c.Request().Header.Get("HX-Request") == "true" {
			return renderPasswordChangeState(c, "Passwords do not match.", "")
		}
		return api_error.NewValidation(c).Msg("Passwords do not match.")
	}

	if l.UserID != c.UserProfile.ID && !c.IsAdmin() {
		if c.Request().Header.Get("HX-Request") == "true" {
			return renderPasswordChangeState(c, "Not authorized.", "")
		}
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Not authorized")
	}

	user, err := mq.FindUserProfile(context.Background(), db.DBob, l.UserID)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" && l.UserID == c.UserProfile.ID {
			return renderPasswordChangeState(c, "Unable to find user.", "")
		}
		return api_error.NewWithError(c, err).Msg("error finding user")
	}

	passwd, err := auth2.NewAuth().CryptPassword(l.Password)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" && l.UserID == c.UserProfile.ID {
			return renderPasswordChangeState(c, "Unable to encrypt password.", "")
		}
		return api_error.NewWithError(c, err).Msg("error encrypting password")
	}
	if err := user.Update(context.Background(), db.DBob, &mq.UserProfileSetter{
		Password:  omit.From(passwd),
		UpdatedAt: omitnull.From(time.Now()),
	}); err != nil {
		if c.Request().Header.Get("HX-Request") == "true" && l.UserID == c.UserProfile.ID {
			return renderPasswordChangeState(c, "Unable to update password.", "")
		}
		return api_error.NewWithError(c, err).Msg("error updating user")
	}

	if c.Request().Header.Get("HX-Request") == "true" && l.UserID == c.UserProfile.ID {
		return renderPasswordChangeState(c, "", "Password changed successfully.")
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, user.ID, true, true))
}

func (m *api) ResetUserPassword(c *mw.UserContext) error {
	type request struct {
		UserID int32 `json:"user_id" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if l.UserID == c.UserProfile.ID {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Please set a new password. You can not reset it.")
	}

	if l.UserID != c.UserProfile.ID && !c.IsAdmin() {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("Not authorized")
	}

	var user *mq.UserProfile
	if errTx := db.DBob.RunInTx(context.Background(), nil, func(ctx context.Context, tx bob.Executor) error {

		user, err = mq.FindUserProfile(ctx, db.DBob, l.UserID)
		if err != nil {
			return fmt.Errorf("error finding user:%w", err)
		}
		if user, err = auth2.NewAuth().ResetPassword(ctx, tx, user); err != nil {
			return fmt.Errorf("error resetting password:%w", err)
		}

		return nil
	}); errTx != nil {
		return api_error.NewWithError(c, errTx).Msg("error ResetUserPassword user password")
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, user.ID, true, true))
}

func (m *api) AddUser(c *mw.UserContext) error {
	type request struct {
		Salutation        string `json:"salutation" validate:"required"`
		Forename          string `json:"forename" validate:"required"`
		Lastname          string `json:"lastname" validate:"required"`
		Email             string `json:"email" validate:"required,email"`
		MustResetPassword bool   `json:"must_reset_password"`
		Disabled          bool   `json:"disabled"`
		EmailValidated    bool   `json:"email_validated"`
		Password          string `json:"password" validate:"required"`
	}
	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	a := auth2.NewAuth()
	exist, err := mq.UserProfiles.Query(mq.SelectWhere.UserProfiles.Email.EQ(l.Email)).Exists(c.Srv.Ctx, c.Srv.DbTx())
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error finding user")
	}
	if exist {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg(Tr(LK_email_already_exists))
	}
	passwd, err := a.CryptPassword(l.Password)
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error encrypting password")
	}

	user, err := a.CreateProfile(c.Srv, l.Email, passwd, l.Salutation, l.Forename, l.Lastname, l.Disabled, l.EmailValidated, false, db.NullVal(c.UserProfile.ID))
	if err != nil {
		return api_error.NewWithError(c, err).Msg("error adding user")
	}

	return c.JSON(http.StatusOK, auth.GetUserDetails(context.Background(), db.DBob, user.ID, true, true))
}
