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
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nakami-lounge-GmbH/ui-components/datatable"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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

// BlacklistAdd nimmt eine einzelne Adresse manuell in die Blacklist auf
// (reason='Manual', source='manual') und liefert die aktualisierte Tabelle
// samt Erfolgs-Toast zurück.
func (m *api) BlacklistAdd(c *mw.UserContext) error {
	if _, err := postmark.AddToBlacklist(c.Request().Context(), c.FormValue("email")); err != nil {
		msg := err.Error()
		switch {
		case errors.Is(err, postmark.ErrBlacklistInvalidEmail):
			msg = i18n.TrLang(c.Lang, i18n.L.Blacklist.InvalidEmailAddress)
		case errors.Is(err, postmark.ErrBlacklistAlreadyExists):
			msg = i18n.TrLang(c.Lang, i18n.L.Blacklist.AddressIsAlreadyBlacklisted)
		}
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), msg)
	}
	rows, state, err := loadBlacklist(c)
	if err != nil {
		return err
	}
	return views.RenderWithToast(c,
		blacklistview.BlacklistPage(c.Lang, rows, state),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Blacklist.AddressAddedToBlacklist),
	)
}

// BlacklistExport liefert die komplette Blacklist als CSV-Download
// (Content-Disposition: attachment). Trennzeichen Semikolon + UTF-8-BOM, damit
// die Datei in (deutschem) Excel per Doppelklick korrekt in Spalten landet.
func (m *api) BlacklistExport(c *mw.UserContext) error {
	ctx := c.Request().Context()
	rows, err := mq.MailBlacklists.Query(
		sm.OrderBy(mq.MailBlacklists.Columns.CreatedAt).Desc(),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}

	filename := "blacklist-" + time.Now().Format("2006-01-02") + ".csv"
	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
	res.Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	res.WriteHeader(http.StatusOK)

	// UTF-8-BOM voranstellen, damit Excel die Kodierung erkennt (Umlaute).
	_, _ = res.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(res)
	w.Comma = ';'

	lang := c.Lang
	_ = w.Write([]string{
		i18n.TrLang(lang, i18n.L.Blacklist.EmailAddress),
		i18n.TrLang(lang, i18n.L.Blacklist.Reason),
		i18n.TrLang(lang, i18n.L.Blacklist.Origin),
		i18n.TrLang(lang, i18n.L.Blacklist.Source),
		i18n.TrLang(lang, i18n.L.Blacklist.BlockedAt),
	})
	for _, b := range rows {
		_ = w.Write([]string{
			b.Email,
			b.Reason,
			b.Origin,
			b.Source,
			b.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	w.Flush()
	return w.Error()
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
