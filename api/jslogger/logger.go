package jslogger

import (
	"app/api/api_error"
	"app/api/validate"
	"app/mw"
	"fmt"
	"log"
	"net/http"
	"time"
)

func JsLog(c *mw.AnonUserContext) error {
	type request struct {
		SecretKey       string    `json:"secret_key" validate:"required"`
		ClientTimestamp time.Time `json:"client_timestamp" validate:"required"`
		Message         string    `json:"message" validate:"required"`
		Type            string    `json:"type"`
		Component       string    `json:"component"`
		LineNr          string    `json:"line_nr"`
		URL             string    `json:"url"`
		Params          string    `json:"params"`
		Browser         string    `json:"browser"`
	}

	l, err := validate.ValidateData[request](c)
	if err != nil {
		return validate.ValidationError(c, err)
	}

	if l.SecretKey != AuthKeyJsLog {
		return api_error.NewWithCode(c, http.StatusBadRequest).Msg("wrong")
	}

	ip := c.Request().RemoteAddr
	referer := c.Request().Referer()
	msg := "JS-Error on client:"
	if c.HasUser {
		msg += fmt.Sprintf(" UserID: <%d>", c.UserProfile.ID)
	}
	log.Printf("%s IP: <%s>, Referer: <%s>, clientTimestamp: <%v>, message:<%s>, Type: <%s>, Component: <%s>, LineNr: <%s>, URL: <%s> Params: <%s> ClientBrowser:<%s>, UserAgent:<%s>\n", msg, ip, referer, l.ClientTimestamp, l.Message, l.Type, l.Component, l.LineNr, l.URL, l.Params, l.Browser, c.Request().UserAgent())

	return c.JSON(http.StatusOK, "{}")
}
