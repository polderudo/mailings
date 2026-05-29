package auth

import (
	"app/api/api_error"
	"app/api/validate"
	authz "app/auth"
	"app/conf"
	"app/constants"
	coreauth "app/core/auth"
	"app/db"
	. "app/i18n"
	"app/mq"
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aarondl/opt/omitnull"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func checkPassword(pwd string, hash string) bool {
	if pwd == "" || hash == "" {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pwd), []byte(hash)); err != nil {
		return false
	}
	return true
}

func shouldUseSecureCookies(c echo.Context) bool {
	return conf.C.UseHTTPs || c.Scheme() == "https"
}

func setAuthJWTCookie(c echo.Context, token string) {
	cookie := new(http.Cookie)
	cookie.Name = echo.HeaderAuthorization
	cookie.Path = "/"
	cookie.Value = token
	cookie.Expires = time.Now().Add(time.Hour * 72)
	cookie.HttpOnly = true
	cookie.Secure = shouldUseSecureCookies(c)
	c.SetCookie(cookie)
}

func clearAuthJWTCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = echo.HeaderAuthorization
	cookie.Path = "/"
	cookie.Value = ""
	cookie.MaxAge = -1
	cookie.HttpOnly = true
	cookie.Secure = shouldUseSecureCookies(c)
	c.SetCookie(cookie)
}

func authLang(c echo.Context) string {
	return mw.ResolveRequestLang(c)
}

func redirectAfterAuth(c echo.Context, location string) error {
	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", location)
		return c.NoContent(http.StatusNoContent)
	}
	return c.Redirect(http.StatusSeeOther, location)
}

func Login(c echo.Context) error {
	type request struct {
		Email    string `form:"email" validate:"required,email"`
		Password string `form:"password" validate:"required"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}
	l.Email = coreauth.TrimmedLower(l.Email)

	cx := context.Background()
	user, err := mq.UserProfiles.Query(mq.SelectWhere.UserProfiles.Email.EQ(l.Email)).One(cx, db.DBob)
	if err != nil {
		if db.IsNoRows(err) {
			return views.Render(c, authshared.Login(TrLang(authLang(c), L.Auth.YouHaveEnteredWrongCredentials), authLang(c)))
		}
		return fmt.Errorf("error reading user by token: %w", err)
	}

	if user.Disabled {
		return views.Render(c, authshared.Login(TrLang(authLang(c), L.Auth.YouHaveEnteredWrongCredentials), authLang(c)))
	}

	if user.MustResetPassword {
		return views.Render(c, authshared.Login(TrLang(authLang(c), L.Auth.YouHaveEnteredWrongCredentials), authLang(c)))
	}

	if !checkPassword(user.Password, l.Password) {
		return views.Render(c, authshared.Login(TrLang(authLang(c), L.Auth.YouHaveEnteredWrongCredentials), authLang(c)))
	}

	token, err := CreateTokenForUser(user)
	if err != nil {
		return fmt.Errorf("error creating token: %w", err)
	}

	if err := user.Update(cx, db.DBob, &mq.UserProfileSetter{
		LastLogin: omitnull.From(time.Now()),
	}); err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	setAuthJWTCookie(c, token)

	return redirectAfterAuth(c, "/")
}

func Logoff(c echo.Context) error {
	clearAuthJWTCookie(c)

	return redirectAfterAuth(c, "/auth/login/")
}

func CreateTokenForUser(user *mq.UserProfile) (string, error) {
	dur := time.Minute * time.Duration(conf.C.JWTTokenDurationMinutes)

	if conf.C.LoginValidForever {
		dur = time.Hour * time.Duration(100_000)
	}
	claims := &mw.JwtCustomClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(conf.C.JWTSecreteKey))
	if err != nil {
		return "", err
	}
	return t, nil
}

func Refresh(ctx *mw.UserContext) error {
	type response struct {
		Token                   string   `json:"token"`
		JWTTokenDurationMinutes int      `json:"jwt_token_duration_minutes"`
		Permissions             []string `json:"permissions"`
		VisibleNavItems         []string `json:"visible_nav_items"`
	}

	user, err := mq.FindUserProfile(context.Background(), db.DBob, ctx.UserProfile.ID)
	if err != nil {
		if db.IsNoRows(err) {
			return api_error.NewWithCode(ctx, http.StatusBadRequest).Msg(Tr(LK_email_not_found))
		}
		return fmt.Errorf("error reading user by token: %w", err)
	}

	if user.Disabled {
		return api_error.NewWithCode(ctx, constants.ErrUserDeactivated).Msg(Tr(LK_user_is_deactivated))
	}

	if user.MustResetPassword {
		return api_error.NewWithCode(ctx, constants.ErrUserNoPassword).Msg(Tr(LK_user_must_change_set_password))
	}

	token, err := CreateTokenForUser(user)
	if err != nil {
		return fmt.Errorf("error creating token: %w", err)
	}

	jwtTokenDurationMinutes := conf.C.JWTTokenDurationMinutes
	if conf.C.LoginValidForever {
		jwtTokenDurationMinutes = 100_000 * 60
	}

	return ctx.JSON(http.StatusOK, response{
		Token:                   token,
		JWTTokenDurationMinutes: jwtTokenDurationMinutes,
		Permissions:             authz.GetUserPermissions(user.ID),
	})
}
