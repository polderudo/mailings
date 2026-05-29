package api

import (
	"app/conf"
	"app/db"
	"app/internals"
	"app/mail"
	"app/mw"
	"app/templates"
	errorsview "app/views/pages/errors"
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/labstack/echo/v4"
)

// HeaderToMap converts a header to an string array
func HeaderToMap(header http.Header) (res map[string]string) {
	ret := make(map[string]string)
	for name, values := range header {
		for _, value := range values {
			ret[name] = value
		}
	}
	return ret
}

// Stack Return the current stactrace
func Stack() string {
	buf := make([]byte, 1<<16)
	runtime.Stack(buf, false)
	return string(bytes.Trim(buf, "\x00"))
}

// HandlePanic general panic handler
func HandlePanic(function string, message string) {
	if r := recover(); r != nil {
		log.Printf("Recovered in <%s>. Message <%s>, Rec:<%v>, Stack:<%v>", function, message, r, Stack())
	}
}

func CustomHTTPErrorHandler(err error, c echo.Context) {
	if err == nil {
		return
	}

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}

	if code == http.StatusNotFound && !c.Response().Committed {
		anonCtx := mw.NewAnonUserContext(c)
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
		c.Response().WriteHeader(http.StatusNotFound)
		if anonCtx.HasUser && anonCtx.UserProfile != nil {
			_ = errorsview.NotFoundPage(anonCtx.UserProfile, anonCtx.Lang).Render(anonCtx.Request().Context(), c.Response())
		} else {
			_ = errorsview.NotFoundGuestPage().Render(anonCtx.Request().Context(), c.Response())
		}
		return
	}

	stack := Stack()
	c.Echo().DefaultHTTPErrorHandler(err, c)

	ignoredErrors := []string{
		"missing or malformed jwt",
		"invalid or expired jwt",
		"Method Not Allowed",
	}

	if code != http.StatusNotFound && code != http.StatusUnauthorized && !db.IsNoRows(err) && !internals.ContainsSubstring(err.Error(), ignoredErrors) {
		c.Logger().Error(err, "\n", stack) //log also the stack
		go func(err error, c echo.Context, stack string) {
			defer HandlePanic("CustomHTTPErrorHandler", "Error on handling panic")

			code := http.StatusInternalServerError
			if he, ok := err.(*echo.HTTPError); ok {
				code = he.Code
			}
			if code != 404 && code != 401 {
				host, _ := os.Hostname()

				safe := strings.Replace(stack, "\n", "<br>", -1)
				safe = strings.Replace(safe, "\t", "&nbsp;&nbsp;", -1)

				data := map[string]interface{}{
					"Err":          err,
					"Code":         code,
					"C":            c,
					"Stack":        template.HTML(safe),
					"Hostname":     host,
					"ClientAdress": internals.GetIP(c),
					"Headers":      HeaderToMap(c.Request().Header),
					"RequestID":    c.Response().Header().Get(echo.HeaderXRequestID),
				}
				errorTemplate := templates.GetHTMLTemplate(templates.TplAppError)

				var s bytes.Buffer
				if err1 := errorTemplate.Execute(&s, data); err1 != nil {
					c.Logger().Error(err1)
				} else {
					go mail.SendMail(context.Background(), db.DBob, conf.C.MailConfig.Sender, conf.C.ApplicationAdmins, nil, nil, "Error on Application ("+conf.C.ApplicationName+")", "", s.String(), "", "")
				}
			}
		}(err, c, stack)
	}
}
