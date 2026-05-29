package api

import (
	"app/api/router"
	"app/conf"
	"app/mw"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"

	_ "net/http/pprof"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4/middleware"
)

var (
	WebLogFile *os.File
)

func StartCoreAPI() error {
	e := echo.New()
	e.HideBanner = true
	router.ERouter = e

	//e.Pre(middleware.AddTrailingSlash())
	e.Use(middleware.RequestID())

	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisableStackAll:   false,
		DisablePrintStack: false,
	}))

	e.Use(mw.WebLog)

	e.HTTPErrorHandler = CustomHTTPErrorHandler

	valid := validator.New(validator.WithRequiredStructEnabled())
	if err := valid.RegisterValidation("eitherrequired", oneFieldSet); err != nil {
		log.Fatalf("could not register validator:%v\n", err)
	}

	//valid.RegisterCustomTypeFunc(ValidateValuer, models.NullHutTypes{})

	e.Validator = &CustomValidator{valid}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3002", "http://localhost:3000", "http://devzone.me:3000", "http://devserver:3000", "http://localhost:3002",
			"https://localhost:8080", "https://localhost:3000", "https://devzone.me:3000", "https://devserver:3000", "https://localhost:3002", "*"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, echo.HeaderAccessControlAllowCredentials, "Cache-Control", "X-Requested-With", "Station"},
		AllowMethods:     []string{echo.GET, echo.HEAD, echo.PUT, echo.POST, echo.DELETE, echo.OPTIONS},
		ExposeHeaders:    []string{echo.HeaderAuthorization, "Content-Disposition", "Station"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) (bool, error) {
			return true, nil
		},
	}))

	jwtConfig := echojwt.Config{
		SigningKey: []byte(conf.C.JWTSecreteKey),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(mw.JwtCustomClaims)
		},
		TokenLookup: "header:Authorization:Bearer ,query:authorization,cookie:Authorization",
		BeforeFunc: func(c echo.Context) {
			return
		},
	}

	CreateCoreRoutes(e, jwtConfig)

	/*for _, ro := range e.Routes() {
		if ro.Method != "CONNECT" && ro.Method != "PROPFIND" && ro.Method != "HEAD" && ro.Method != "PATCH" && ro.Method != "OPTIONS" &&
			ro.Method != "REPORT" && ro.Method != "TRACE" && !strings.Contains(ro.Name, "glob..") &&
			ro.Method != "echo_route_not_found" {
			slog.Info("", slog.String("method", ro.Method), slog.String("path", ro.Path), slog.String("name", ro.Name))
		}
	}*/

	srv := ""
	if conf.C.BindLocalhost {
		srv = "localhost"
	}

	//give the tcp servers 5 seconds time to connect
	time.Sleep(2 * time.Second)
	s := fmt.Sprintf("%s:%d", srv, conf.C.Port)
	log.Printf("Starting API on %s\n", s)
	if !conf.C.UseHTTPs {
		e.Logger.Fatal(e.Start(s))
	} else {
		if err := e.StartTLS(s, conf.C.HTTPSCertFile, conf.C.HTTPSKeyFile); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}

	if WebLogFile != nil {
		WebLogFile.Close()
	}

	return nil
}
