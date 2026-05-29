package pages

import (
	"app/api/router"
	"app/db"
	"app/mw"
	"app/views"
	authshared "app/views/pages/auth/shared"
	usersview "app/views/pages/main/users"

	"github.com/nakami-lounge-GmbH/ui-components/datatable"
)

func UsersPage(c *mw.AnonUserContext) error {
	if !c.HasUser {
		return views.Render(c, authshared.Login("", c.Lang))
	}

	bind := db.BindPaginated(c, 50, func(b *db.FilterBinder, criteria *db.UserProfileCriteria) {
		b.String("full_name", &criteria.FullName)
		b.String("status", &criteria.Status)
	})
	rows, total, err := db.QueryUserProfileRows(c.Request().Context(), bind.Pagination, bind.Criteria)
	if err != nil {
		return err
	}
	state := datatable.TableState{
		Page:      bind.Pagination.Page,
		Count:     bind.Pagination.Count,
		Total:     total,
		SortField: bind.SortField,
		SortDesc:  bind.SortDesc,
		Filters:   bind.Filters,
		Endpoint:  router.Reverse(router.Users.List),
		Target:    "#layout-content-body",
	}
	if isHXRequest(c) {
		return views.Render(c, usersview.UsersPage(c.Lang, rows, state))
	}
	return views.Render(c, usersview.Users(c.UserProfile, c.Lang, rows, state))
}
