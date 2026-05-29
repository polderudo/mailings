package auth

import (
	"app/api/api_error"
	"app/api/router"
	"app/api/validate"
	"app/conf"
	"app/constants"
	coreauth "app/core/auth"
	"app/core/auth/values"
	"app/db"
	. "app/i18n"
	"app/mail"
	"app/mq"
	"app/mw"
	"app/templates"
	"app/views"
	authshared "app/views/pages/auth/shared"
	settingsview "app/views/pages/main/settings"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/labstack/echo/v4"
	"github.com/nakami-lounge-GmbH/tools/helpers"
	"github.com/stephenafamo/bob"
)

func getUserWithValidToken(ctx context.Context, tx bob.Executor, tokenName string, token string) (*mq.UserProfile, *mq.Token, error) {
	user, err := mq.UserProfiles.Query(
		mq.SelectJoins.UserProfiles.InnerJoin.UserTokens,
		mq.SelectWhere.Tokens.TokenName.EQ(tokenName),
		mq.SelectWhere.Tokens.TokenValue.EQ(token),
		mq.SelectWhere.Tokens.ValidTo.GTE(time.Now()),
		mq.SelectWhere.Tokens.ValidatedAt.IsNull(),
	).One(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("error selecting user: %w", err)
	}

	t, err := mq.Tokens.Query(
		mq.SelectWhere.Tokens.UserID.EQ(user.ID),
		mq.SelectWhere.Tokens.TokenName.EQ(tokenName),
		mq.SelectWhere.Tokens.TokenValue.EQ(token),
		mq.SelectWhere.Tokens.ValidTo.GTE(time.Now()),
		mq.SelectWhere.Tokens.ValidatedAt.IsNull(),
	).One(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("error selecting token: %w", err)
	}

	return user, t, nil
}

func CheckUserToken(c echo.Context) error {
	type request struct {
		Token     string `json:"token" query:"token" validate:"required"`
		TokenName string `json:"token_name" query:"token_name" validate:"required,oneof=UserPassword EmailChange"`
	}

	type result struct {
		ID       int32  `boil:"id" json:"id" toml:"id" yaml:"id"`
		Email    string `boil:"email" json:"email" toml:"email" yaml:"email"`
		Forename string `boil:"forename" json:"forename" toml:"forename" yaml:"forename"`
		Lastname string `boil:"lastname" json:"lastname" toml:"lastname" yaml:"lastname"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	ut, _, err := getUserWithValidToken(context.Background(), db.DBob, l.TokenName, l.Token)
	if err != nil {
		if db.IsNoRows(err) {
			return api_error.NewWithCode(c, constants.ErrTokenNotExists).Msg("The specified user does not exist, or the valid-date of the link expired")
		}
		return api_error.NewWithError(c, err).Msg("The specified user does not exist, or the valid-date of the link expired")
	}

	return c.JSON(http.StatusOK, &result{
		ID:       ut.ID,
		Email:    ut.Email,
		Forename: ut.Forename,
		Lastname: ut.Lastname,
	})
}

func SetPasswordForCurrentUser(c *mw.UserContext) error {
	type request struct {
		Password        string `form:"password" validate:"required,min=8"`
		ConfirmPassword string `form:"confirm_password" validate:"required"`
	}

	user := c.UserProfile
	if user == nil {
		return views.Render(c, settingsview.ChangePasswordState(c.UserProfile, "Unable to read current user.", "", c.Lang))
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return views.Render(c, settingsview.ChangePasswordState(user, "Please enter a valid password (min 8 chars).", "", c.Lang))
	}

	l.Password = strings.Trim(l.Password, "\r\n\t ")
	l.ConfirmPassword = strings.Trim(l.ConfirmPassword, "\r\n\t ")

	if len(l.Password) < 8 {
		return views.Render(c, settingsview.ChangePasswordState(user, "The password must be at least 8 character long", "", c.Lang))
	}

	if l.Password != l.ConfirmPassword {
		return views.Render(c, settingsview.ChangePasswordState(user, "Passwords do not match.", "", c.Lang))
	}

	passw, err := coreauth.NewAuth().CryptPassword(l.Password)
	if err != nil {
		return views.Render(c, settingsview.ChangePasswordState(user, "Unable to encrypt password.", "", c.Lang))
	}

	if err := user.Update(context.Background(), db.DBob, &mq.UserProfileSetter{
		Password:          omit.From(passw),
		MustResetPassword: omit.From(false),
		UpdatedAt:         omitnull.From(time.Now()),
	}); err != nil {
		return views.Render(c, settingsview.ChangePasswordState(user, "Unable to update password.", "", c.Lang))
	}

	return views.Render(c, settingsview.ChangePasswordState(user, "", "Password updated successfully.", c.Lang))
}

type setPasswordByTokenInput struct {
	Token              string
	TokenName          string
	Password           string
	RequireUserMatch   bool
	ExpectedUserID     int32
	ExpectedUserEmail  string
}

func setPasswordByToken(ctx context.Context, tx bob.Executor, input setPasswordByTokenInput) (*mq.UserProfile, error) {
	user, token, err := getUserWithValidToken(ctx, tx, input.TokenName, input.Token)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}

	if input.RequireUserMatch && (user.ID != input.ExpectedUserID || user.Email != input.ExpectedUserEmail) {
		return nil, fmt.Errorf("user data does not match")
	}

	pwd, err := coreauth.NewAuth().CryptPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("error crypting password: %w", err)
	}

	us := &mq.UserProfileSetter{
		EmailValidated:    omit.From(true),
		MustResetPassword: omit.From(false),
		Password:          omit.From(pwd),
		Disabled:          omit.From(false),
		UpdatedAt:         omitnull.From(time.Now()),
	}
	if err = user.Update(ctx, tx, us); err != nil {
		return nil, fmt.Errorf("error updating user: %w", err)
	}

	if err = token.Update(ctx, tx, &mq.TokenSetter{
		ValidatedAt: omitnull.From(time.Now()),
		TokenValue:  omit.From(token.TokenValue + "validated"),
		ValidTo:     omit.From(time.Now().Add(-1000 * time.Hour)),
	}); err != nil {
		return nil, fmt.Errorf("error updating token: %w", err)
	}

	return user, nil
}

func sendPasswordChangedNotification(user *mq.UserProfile) error {
	data := struct {
		User    *mq.UserProfile
		WebLink string
	}{
		user,
		conf.C.WebURL.WebURLLogin,
	}

	body, err := templates.ExecuteTextTemplate(templates.TplPasswordChanged, data)
	if err != nil {
		return fmt.Errorf("error executing mail template: %w", err)
	}

	subject := Tr(LK_password_change_for_s, conf.C.ApplicationName)
	go func(uu *mq.UserProfile, ss string, bb string) {
		if err := mail.SendMail(context.Background(), db.DBob, conf.C.MailConfig.Sender, []string{uu.Email}, nil, nil, ss, bb, "", "", ""); err != nil {
			log.Printf("Error sonding mail: %v", err)
		}
	}(user, subject, body)

	return nil
}

func SetPasswordByToken(c echo.Context) error {
	type request struct {
		Token     string `json:"token" validate:"required"`
		TokenName string `json:"token_name" validate:"required,oneof=UserPassword"`
		UserID    int32  `json:"user_id" validate:"required"`
		Email     string `boil:"email" validate:"required"`
		Password  string `json:"password" validate:"required,min=8"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	l.Password = strings.Trim(l.Password, "\r\n\t ")
	if len(l.Password) < 8 {
		return api_error.NewWithCode(c, constants.ErrToShort).Msg("The password must be at least 8 character long")
	}

	var user *mq.UserProfile
	if errTx := db.DBob.RunInTx(context.Background(), nil, func(ctx context.Context, tx bob.Executor) error {
		user, err = setPasswordByToken(ctx, tx, setPasswordByTokenInput{
			Token:             l.Token,
			TokenName:         l.TokenName,
			Password:          l.Password,
			RequireUserMatch:  true,
			ExpectedUserID:    l.UserID,
			ExpectedUserEmail: l.Email,
		})
		return err
	}); errTx != nil {
		return api_error.NewWithError(c, errTx).Msg("error SetPasswordByToken")
	}

	if err = sendPasswordChangedNotification(user); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, "{}")
}

func SetPasswordByTokenPage(c echo.Context) error {
	type request struct {
		Password        string `form:"password" validate:"required,min=8"`
		ConfirmPassword string `form:"confirm_password" validate:"required"`
	}

	tokenValue := strings.Trim(c.Param("token"), "\r\n\t ")
	lang := authLang(c)
	if tokenValue == "" {
		return views.Render(c, authshared.Login("", lang))
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return views.Render(c, authshared.PasswordCreate(tokenValue, TrLang(lang, L.Auth.PleaseEnterAValidPasswordMin8Chars), lang))
	}

	l.Password = strings.Trim(l.Password, "\r\n\t ")
	l.ConfirmPassword = strings.Trim(l.ConfirmPassword, "\r\n\t ")
	if len(l.Password) < 8 {
		return views.Render(c, authshared.PasswordCreate(tokenValue, TrLang(lang, L.Auth.MustBeAtLeast8CharactersLong), lang))
	}
	if l.Password != l.ConfirmPassword {
		return views.Render(c, authshared.PasswordCreate(tokenValue, TrLang(lang, L.Auth.PasswordsDoNotMatch), lang))
	}

	var user *mq.UserProfile
	if errTx := db.DBob.RunInTx(context.Background(), nil, func(ctx context.Context, tx bob.Executor) error {
		user, err = setPasswordByToken(ctx, tx, setPasswordByTokenInput{
			Token:     tokenValue,
			TokenName: values.TokenNameUserPassword,
			Password:  l.Password,
		})
		return err
	}); errTx != nil {
		return views.Render(c, authshared.PasswordCreate(tokenValue, TrLang(lang, L.Auth.TheResetLinkIsInvalidOrExpired), lang))
	}

	if err = sendPasswordChangedNotification(user); err != nil {
		return err
	}

	loginURL := router.Reverse(router.Anon.LoginPage)
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", loginURL)
		return c.NoContent(http.StatusNoContent)
	}
	return c.Redirect(http.StatusSeeOther, loginURL)
}

func ForgotPassword(c echo.Context) error {
	type request struct {
		Email string `json:"email" form:"email" validate:"required,email"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	l.Email = coreauth.TrimmedLower(l.Email)

	if errTx := db.DBob.RunInTx(context.Background(), nil, func(ctx context.Context, tx bob.Executor) error {
		user, err := mq.UserProfiles.Query(
			mq.SelectWhere.UserProfiles.Email.EQ(l.Email),
		).One(ctx, tx)
		if err != nil {
			// Do not leak whether a user exists.
			if db.IsNoRows(err) {
				return nil
			}
			return fmt.Errorf("error selecting user: %w", err)
		}

		if err = user.Update(ctx, tx, &mq.UserProfileSetter{
			MustResetPassword: omit.From(true),
			UpdatedAt:         omitnull.From(time.Now()),
		}); err != nil {
			return fmt.Errorf("error updating user: %w", err)
		}

		validTo := time.Now().Add(values.TokenPasswordsetValid)

		tt := helpers.RandomString(10)
		webLink := fmt.Sprintf(conf.C.WebURL.CreatePassword, tt)

		tokenSetter := &mq.TokenSetter{
			UserID:     omitnull.From(user.ID),
			TokenName:  omit.From(values.TokenNameUserPassword),
			TokenValue: omit.From(tt),
			TokenURI:   omit.From(webLink),
			ValidTo:    omit.From(validTo),
		}

		_, err = mq.Tokens.Insert(tokenSetter).One(ctx, tx)
		if err != nil {
			return fmt.Errorf("error inserting token: %w", err)
		}

		data := struct {
			User    *mq.UserProfile
			WebLink string
			ValidTo time.Time
		}{
			user,
			webLink,
			validTo,
		}

		body, err := templates.ExecuteTextTemplate(templates.TplPasswordChangeTokenMail, data)
		if err != nil {
			return fmt.Errorf("error adding token: %w", err)
		}

		subject := Tr(LK_password_reset_for_s, conf.C.ApplicationName)
		if err := mail.SendMail(ctx, tx, conf.C.MailConfig.Sender, []string{user.Email}, nil, nil, subject, body, "", "", ""); err != nil {
			return fmt.Errorf("error sending mail: %w", err)
		}

		return nil
	}); errTx != nil {
		return api_error.NewWithError(c, errTx).Msg("error ForgotPassword")
	}

	// If called from the web UI (HTML form), render HTML so the user sees feedback.
	// Otherwise (API-style JSON request), keep the existing JSON response.
	if strings.Contains(c.Request().Header.Get(echo.HeaderContentType), echo.MIMEApplicationJSON) {
		return c.JSON(http.StatusOK, "{}")
	}
	return views.Render(c, authshared.ForgotPassword(
		"",
		TrLang(authLang(c), L.Auth.IfTheEmailExistsInOurSystemYoulKpnwgmvv),
		authLang(c),
	))
}