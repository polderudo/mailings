package api

import (
	"app/api/anon"
	"app/api/jslogger"
	authApi "app/core/auth/api"
	listsApi "app/core/lists/api"
	mailingsApi "app/core/mailings/api"
	templatesApi "app/core/templates/api"

	"app/mw"
	"net/http"
	"strings"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	templuiutils "github.com/templui/templui/utils"
)

func CreateCoreRoutes(e *echo.Echo, config echojwt.Config) {
	e.Pre(middleware.AddTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return strings.HasPrefix(path, "/static/") || strings.Contains(path, ".")
		},
	}))
	e.Static("/static", "assets")

	templuiMux := http.NewServeMux()
	templuiutils.SetupScriptRoutes(templuiMux, true)
	e.GET("/templui/js/*", echo.WrapHandler(templuiMux))
	e.Add("GET", "/.well-known/appspecific/com.chrome.devtools.json", func(c echo.Context) error {
		return c.String(200, `{"devtools_frontend_url": "/devtools"}`)
	})
	e.Add("GET", "/favicon.ico", func(c echo.Context) error {
		return c.String(200, `{}`)
	})

	rApi := e.Group("")
	rApi.Use(mw.SetAnonUserContext)
	rAnon := mw.NewAnonGroup(rApi)

	rEchoUser := rApi.Group("/a")
	rEchoUser.Use(echojwt.WithConfig(config))
	rUser := mw.NewUserGroup(rEchoUser)

	rUser.EchoGroup.Use(mw.SetUserContext)

	anon.CreateRoutes(rAnon, rUser)

	rAdmin := rUser.Group("/ad")
	rAdmin.EchoGroup.Use(mw.SetAdminContext)

	rJsLog := rAnon.Group("/jslog")
	jslogger.CreateRoutes(rJsLog)

	authApi.CreateRoutes(rUser, rAdmin)
	templatesApi.CreateRoutes(rUser)
	listsApi.CreateRoutes(rUser)
	mailingsApi.CreateRoutes(rUser)

	return
}
