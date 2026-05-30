package api

import (
	"app/api/anon"
	"app/api/jslogger"
	"app/assets"
	"app/conf"
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
	e.GET("/static/*", staticHandler())

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

// staticHandler serves everything under /static/. By default assets are read
// from the embedded filesystem (assets.FS) so the binary is self-contained.
// When conf.C.ServeAssetsFromDisk is set (dev), the embed base is swapped for a
// disk-rooted FS so CSS/JS/img edits are live without rebuilding.
func staticHandler() echo.HandlerFunc {
	if conf.C != nil && conf.C.ServeAssetsFromDisk {
		assets.UseDiskBase("assets")
	}
	fileServer := http.FileServer(http.FS(assets.FS))

	return func(c echo.Context) error {
		// output.css is rebuilt on every deploy; prevent browsers from serving a stale version.
		if strings.HasSuffix(c.Request().URL.Path, "/output.css") {
			c.Response().Header().Set("Cache-Control", "no-store")
		}
		http.StripPrefix("/static/", fileServer).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
