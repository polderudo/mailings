package pages

import (
	"app/db"
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	usersview "app/views/pages/main/users"

	globaltable "github.com/nakami-lounge-GmbH/ui-components/table"
)

func UsersPage(c *mw.AnonUserContext) error {
	if c.HasUser {
		request := globaltable.ReadRequest(c.Request(), usersview.CurrentUsersTableID)
		sorts := make(db.SortParams, len(request.Pagination.Sorts))
		for i, s := range request.Pagination.Sorts {
			sorts[i] = db.SortParam{Field: s.Field, IsDesc: s.IsDesc}
		}
		if v := c.QueryParam("filter_status"); v != "" {
			request.FilterCriteria["filterStatus"] = v
		}
		rows, total, err := db.LoadUserProfileTableRows(c.Request().Context(), db.FilterRequest[map[string]string]{
			P: db.PaginationData{
				Page:  request.Pagination.Page,
				Count: request.Pagination.Count,
				Sorts: sorts,
			},
			FilterCriteria: request.FilterCriteria,
		})
		if err != nil {
			return err
		}

		filterData := usersview.UsersFilterData{
			SelectedStatus: c.QueryParam("filter_status"),
		}

		if isHXRequest(c) {
			if c.Request().Header.Get("HX-Target") == "users-table" {
				return views.Render(c, usersview.CurrentUsersTable(c.Lang, rows, request, total))
			}
			return views.Render(c, usersview.UsersPage(c.Lang, rows, request, total, filterData))
		}
		return views.Render(c, usersview.Users(c.UserProfile, c.Lang, rows, request, total, filterData))
	}

	return views.Render(c, authshared.Login("", c.Lang))
}
