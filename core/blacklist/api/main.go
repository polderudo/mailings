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
	blacklistApi = &api{
		DbCtx: context.Background(),
	}
)

func CreateRoutes(rUser *mw.UserGroup) {
	// GET + POST auf /a/blacklist/: GET = Page, POST = datatable Filter/Sort/Page.
	rUser.AddNamedRoute(router.Blacklist.List, "/blacklist/", blacklistApi.BlacklistPage, http.MethodGet, http.MethodPost)
	// Manuelles Anstoßen des Postmark-Suppression-Imports.
	rUser.AddNamedRoute(router.Blacklist.Sync, "/blacklist/sync/", blacklistApi.BlacklistSync, http.MethodPost)
	// Einzelne Adresse wieder aus der Blacklist nehmen (reaktiviert in Postmark).
	rUser.AddNamedRoute(router.Blacklist.Delete, "/blacklist/:id/", blacklistApi.BlacklistDelete, http.MethodDelete)
}

func isHXRequest(c *mw.UserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
