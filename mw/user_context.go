package mw

import (
	"app/auth"
	"app/db"
	"app/i18n"
	"app/mq"
	"context"
	"net/http"

	srvCtx "app/srvCtx"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// JwtCustomClaims are custom claims extending default ones.
type JwtCustomClaims struct {
	jwt.RegisteredClaims
	UserID int32 `json:"user_id"`
}

// UserContext is a custom EchoContext that also holds the user object
type UserContext struct {
	echo.Context
	*mq.UserProfile
	Lang string
	Srv  *srvCtx.BobContext
}

func (u *UserContext) Error(err error) {
	log.Errorf("error on UserContext:%v", err)
}

func (u *UserContext) setUserData(ctx echo.Context, token *jwt.Token) *UserContext {
	lang := ResolveRequestLang(ctx)
	ctx.SetRequest(ctx.Request().WithContext(i18n.WithLang(ctx.Request().Context(), lang)))
	u.Context = ctx
	u.Lang = lang
	u.Srv = srvCtx.NewEchoBobCtx(ctx)

	claims := token.Claims.(*JwtCustomClaims)

	user, err := mq.FindUserProfile(context.Background(), db.DBob, claims.UserID)

	if err != nil {
		log.Printf("error reading user profile %v\n", err)
		return nil
	}

	if user.Disabled {
		log.Printf("disabled\n")
		return nil
	}

	if user.MustResetPassword {
		log.Printf("must change password\n")
		return nil
	}

	u.UserProfile = user

	return u
}

func NewUserContext(ctx echo.Context) *UserContext {
	tt := ctx.Get("user")
	if tt == nil {
		log.Printf("no user on request\n")
		return nil
	}

	token := tt.(*jwt.Token)
	if token == nil {
		log.Printf("no token on request\n")
		return nil
	}
	ret := &UserContext{}
	ret.setUserData(ctx, token)

	return ret
}

func UserHandler(next func(*UserContext) error) echo.HandlerFunc {
	return func(c echo.Context) error {

		cc, ok := c.(*UserContext)
		if !ok {
			log.Errorf("UserHandler:: Error converting to *UserContext")
			return echo.NewHTTPError(http.StatusUnauthorized, "error reading user rights")
		}
		return next(cc)
	}
}

// SetUserContext sets the user into the context
func SetUserContext(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := NewUserContext(c)
		if cc != nil {
			return f(cc)
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "error setting user context")
	}
}

func (u *UserContext) IsAdmin() bool {
	if u.UserProfile == nil {
		return false
	}
	return auth.IsAdmin(u.ID)
}

func (u *UserContext) IsSuperAdmin() bool {
	if u.UserProfile == nil {
		return false
	}
	return auth.IsSuperAdmin(u.ID)
}

// RequirePermissions takes a handler function and required permissions as variadic parameters.
// It returns a new handler function that checks permissions before calling the original handler.
func RequirePermissions(next func(*UserContext) error, requiredPermissions ...string) func(*UserContext) error {
	return func(c *UserContext) error {
		if c == nil || c.UserProfile == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "User context not available")
		}

		if c.IsAdmin() || c.IsSuperAdmin() {
			return next(c)
		}

		// Get user's permissions
		userPermissions := auth.GetUserPermissions(c.ID)

		// Create a map for faster lookup of user permissions
		userPermMap := make(map[string]bool)
		for _, perm := range userPermissions {
			userPermMap[perm] = true
		}

		// Check if user has all required permissions
		for _, requiredPerm := range requiredPermissions {
			if !userPermMap[requiredPerm] {
				log.Printf("User %d missing required permission: %s\n", c.ID, requiredPerm)
				return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
			}
		}

		// User has all required permissions, call the original handler
		return next(c)
	}
}

// RequirePermission is a convenience function for single permission check
func RequirePermission(next func(*UserContext) error, permission string) func(*UserContext) error {
	return RequirePermissions(next, permission)
}
