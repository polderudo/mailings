package api

import (
	"app/api/router"
	"app/mw"
	"context"
	"net/http"
)

type api struct {
	DbCtx context.Context
}

var (
	mailingsApi = &api{
		DbCtx: context.Background(),
	}
)

func CreateRoutes(rUser *mw.UserGroup) {
	rUser.AddNamedRoute(router.Mailings.List, "/mailings/", mailingsApi.MailingsPage, http.MethodGet)
	rUser.AddNamedRoute(router.Mailings.New, "/mailings/new/", mailingsApi.MailingNewPage, http.MethodGet)
	rUser.AddNamedRoute(router.Mailings.Detail, "/mailings/:id/", mailingsApi.MailingDetailPage, http.MethodGet)
	rUser.AddNamedRoute(router.Mailings.Create, "/mailings/", mailingsApi.MailingCreate, http.MethodPost)
	rUser.AddNamedRoute(router.Mailings.Start, "/mailings/:id/start/", mailingsApi.MailingStart, http.MethodPost)
}

func isHXRequest(c *mw.UserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
