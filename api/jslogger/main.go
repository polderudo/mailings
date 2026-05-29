package jslogger

import (
	"app/api/router"
	"app/mw"
	"net/http"
)

const (
	AuthKeyJsLog = "lfdksjfdsopi93kfdnsdk§gfrsd"
)

func CreateRoutes(rAnon *mw.AnonGroup) {
	rAnon.AddNamedAnonRoute(router.JsLog.Submit, "/", JsLog, http.MethodPost)
	rAnon.AddNamedAnonRoute(router.JsLog.Test, "/test/", Test, http.MethodPost)
	rAnon.AddNamedAnonRoute(router.JsLog.Info, "/info/", Info, http.MethodGet)
}
