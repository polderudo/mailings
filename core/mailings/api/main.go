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
	// GET + POST auf /a/mailings/: GET = Page, POST = datatable Filter.
	// Create wandert auf /a/mailings/new/ um die HTTP-Method-Kollision aufzulösen.
	rUser.AddNamedRoute(router.Mailings.List, "/mailings/", mailingsApi.MailingsPage, http.MethodGet, http.MethodPost)
	rUser.AddNamedRoute(router.Mailings.New, "/mailings/new/", mailingsApi.MailingNewPage, http.MethodGet)
	rUser.AddNamedRoute(router.Mailings.Create, "/mailings/new/", mailingsApi.MailingCreate, http.MethodPost)
	rUser.AddNamedRoute(router.Mailings.Detail, "/mailings/:id/", mailingsApi.MailingDetailPage, http.MethodGet)
	rUser.AddNamedRoute(router.Mailings.Start, "/mailings/:id/start/", mailingsApi.MailingStart, http.MethodPost)
	rUser.AddNamedRoute(router.Mailings.Resend, "/mailings/:id/resend/", mailingsApi.MailingResend, http.MethodPost)
	rUser.AddNamedRoute(router.Mailings.Archive, "/mailings/:id/archive/", mailingsApi.MailingArchive, http.MethodPost)
}

func isHXRequest(c *mw.UserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
