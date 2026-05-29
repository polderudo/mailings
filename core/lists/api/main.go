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
	listsApi = &api{
		DbCtx: context.Background(),
	}
)

func CreateRoutes(rUser *mw.UserGroup) {
	rUser.AddNamedRoute(router.Lists.List, "/lists/", listsApi.ListsPage, http.MethodGet)
	rUser.AddNamedRoute(router.Lists.New, "/lists/new/", listsApi.ListNewPage, http.MethodGet)
	rUser.AddNamedRoute(router.Lists.Detail, "/lists/:id/", listsApi.ListDetailPage, http.MethodGet)
	rUser.AddNamedRoute(router.Lists.Create, "/lists/", listsApi.ListCreate, http.MethodPost)
	rUser.AddNamedRoute(router.Lists.Update, "/lists/:id/", listsApi.ListUpdate, http.MethodPost)
	rUser.AddNamedRoute(router.Lists.Import, "/lists/:id/import/", listsApi.ListImport, http.MethodPost)
	rUser.AddNamedRoute(router.Lists.Delete, "/lists/:id/", listsApi.ListDelete, http.MethodDelete)
	rUser.AddNamedRoute(router.Lists.DeleteRecipient, "/lists/:id/recipients/:rec_id/", listsApi.ListRecipientDelete, http.MethodDelete)
}

func isHXRequest(c *mw.UserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
