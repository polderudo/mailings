package api

import (
	"app/api/router"
	"app/db"
	"app/i18n"
	"app/mail/postmark"
	"app/mq"
	"app/mw"
	"app/views"
	blacklistview "app/views/pages/main/blacklist"
	"fmt"
	"strconv"

	"github.com/nakami-lounge-GmbH/ui-components/datatable"
)

// loadBlacklist liest Filter/Pagination aus dem Request, fragt die Blacklist ab
// und baut den datatable-State. Wird von Page- und Sync-Handler geteilt.
func loadBlacklist(c *mw.UserContext) ([]*mq.MailBlacklist, datatable.TableState, error) {
	bind := db.BindPaginated(c, 50, func(b *db.FilterBinder, criteria *db.MailBlacklistCriteria) {
		b.String("email", &criteria.Email)
		b.String("reason", &criteria.Reason)
	})
	rows, total, err := db.QueryMailBlacklistRows(c.Request().Context(), bind.Pagination, bind.Criteria)
	if err != nil {
		return nil, datatable.TableState{}, err
	}
	state := datatable.TableState{
		Page:      bind.Pagination.Page,
		Count:     bind.Pagination.Count,
		Total:     total,
		SortField: bind.SortField,
		SortDesc:  bind.SortDesc,
		Filters:   bind.Filters,
		Endpoint:  router.Reverse(router.Blacklist.List),
		Target:    "#layout-content-body",
	}
	return rows, state, nil
}

func (m *api) BlacklistPage(c *mw.UserContext) error {
	rows, state, err := loadBlacklist(c)
	if err != nil {
		return err
	}
	if isHXRequest(c) {
		return views.Render(c, blacklistview.BlacklistPage(c.Lang, rows, state))
	}
	return views.Render(c, blacklistview.Blacklist(c.UserProfile, c.Lang, rows, state))
}

// BlacklistSync stößt den Postmark-Suppression-Import manuell an und liefert die
// aktualisierte Tabelle samt Erfolgs-Toast zurück.
func (m *api) BlacklistSync(c *mw.UserContext) error {
	res, err := postmark.SyncSuppressions(c.Request().Context())
	if err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	rows, state, err := loadBlacklist(c)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%s (+%d / -%d, %d %s)",
		i18n.TrLang(c.Lang, i18n.L.Blacklist.BlacklistSynchronized),
		res.Added, res.Removed, res.Total,
		i18n.TrLang(c.Lang, i18n.L.Blacklist.BlockedAddresses),
	)
	return views.RenderWithToast(c,
		blacklistview.BlacklistPage(c.Lang, rows, state),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		msg,
	)
}

// BlacklistDelete nimmt eine einzelne Adresse wieder aus der Blacklist
// (reaktiviert sie in Postmark) und liefert die aktualisierte Tabelle zurück.
func (m *api) BlacklistDelete(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid blacklist id: %w", err)
	}
	if err := postmark.RemoveFromBlacklist(c.Request().Context(), int32(id)); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	rows, state, err := loadBlacklist(c)
	if err != nil {
		return err
	}
	return views.RenderWithToast(c,
		blacklistview.BlacklistPage(c.Lang, rows, state),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Blacklist.AddressRemovedFromBlacklist),
	)
}
