package mw

import (
	"app/conf"
	"app/i18n"
	"errors"
	"net/http"

	srvCtx "app/srvCtx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// AnonUserContext is a custom EchoContext that also holds the user object if present
type AnonUserContext struct {
	*UserContext
	HasUser bool
}

func (u AnonUserContext) Error(err error) {
	log.Errorf("error on AnonUserContext:%v", err)
}

func NewAnonUserContext(ctx echo.Context) *AnonUserContext {
	lang := ResolveRequestLang(ctx)
	ctx.SetRequest(ctx.Request().WithContext(i18n.WithLang(ctx.Request().Context(), lang)))
	baseContext := &UserContext{
		Context: ctx,
		Lang:    lang,
		Srv:     srvCtx.NewEchoBobCtx(ctx),
	}

	//we try to read the token from the cookie
	cookie, err := ctx.Cookie(echo.HeaderAuthorization)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		log.Errorf("error reading cookie: %v", err)
		return &AnonUserContext{UserContext: baseContext, HasUser: false}
	}

	if cookie != nil {
		claims := &JwtCustomClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(conf.C.JWTSecreteKey), nil
		})
		if err != nil {
			log.Errorf("error parsing jwt token: %v", err)
			return &AnonUserContext{UserContext: baseContext, HasUser: false}
		}

		if token != nil && claims != nil {
			userContext := &UserContext{}
			userContext.setUserData(ctx, token)

			ret := &AnonUserContext{
				UserContext: userContext,
				HasUser:     true,
			}

			return ret
		}
	}

	return &AnonUserContext{UserContext: baseContext, HasUser: false}
}

func AnonUserHandler(next func(*AnonUserContext) error) echo.HandlerFunc {
	return func(c echo.Context) error {

		cc, ok := c.(*AnonUserContext)
		if !ok {
			log.Errorf("AnonUserHandler:: Error converting to *AnonUserContext")
			return echo.NewHTTPError(http.StatusUnauthorized, "error reading user rights")
		}
		return next(cc)
	}
}

// SetAnonUserContext sets the user into the context
func SetAnonUserContext(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := NewAnonUserContext(c)
		if cc != nil {
			return f(cc)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "error setting anon user context")
	}
}
