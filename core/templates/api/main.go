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
	templatesApi = &api{
		DbCtx: context.Background(),
	}
)

func CreateRoutes(rUser *mw.UserGroup) {
	// GET + POST auf /a/templates/: GET = Page, POST = datatable Filter.
	// Create wandert auf /a/templates/new/ um die HTTP-Method-Kollision aufzulösen.
	rUser.AddNamedRoute(router.Templates.List, "/templates/", templatesApi.TemplatesPage, http.MethodGet, http.MethodPost)
	rUser.AddNamedRoute(router.Templates.New, "/templates/new/", templatesApi.TemplateNewPage, http.MethodGet)
	rUser.AddNamedRoute(router.Templates.Create, "/templates/new/", templatesApi.TemplateCreate, http.MethodPost)
	rUser.AddNamedRoute(router.Templates.Detail, "/templates/:id/", templatesApi.TemplateDetailPage, http.MethodGet)
	rUser.AddNamedRoute(router.Templates.Update, "/templates/:id/", templatesApi.TemplateUpdate, http.MethodPost)
	rUser.AddNamedRoute(router.Templates.Delete, "/templates/:id/", templatesApi.TemplateDelete, http.MethodDelete)
}

func isHXRequest(c *mw.UserContext) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}
