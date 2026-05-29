package api

import (
	"app/api/router"
	"app/db"
	"app/i18n"
	"app/mq"
	"app/mw"
	"app/views"
	listsview "app/views/pages/main/lists"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/nakami-lounge-GmbH/tools/importer"
	"github.com/nakami-lounge-GmbH/ui-components/datatable"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// defaultPageSize ist die Default-Page-Size, wenn die UI keine count-Param
// mitschickt. Wert wie in mesa2.
const defaultPageSize int64 = 50

// excelRecipientRow ist das Importer-Schema für eine Zeile Empfänger.
// Die `header`-Tags müssen mit den Excel-Spaltenüberschriften übereinstimmen
// (Case-insensitiv, getrimmt).
type excelRecipientRow struct {
	Email    string `header:"Email" validate:"required,email"`
	Forename string `header:"Vorname"`
	Lastname string `header:"Nachname"`
}

func (m *api) ListsPage(c *mw.UserContext) error {
	bind := db.BindPaginated(c, defaultPageSize, func(b *db.FilterBinder, criteria *db.MailListCriteria) {
		b.String("name", &criteria.Name)
		b.String("description", &criteria.Description)
	})
	rows, total, err := db.QueryMailListRows(c.Request().Context(), bind.Pagination, bind.Criteria)
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
		Endpoint:  router.Reverse(router.Lists.List),
		Target:    "#layout-content-body",
	}
	if isHXRequest(c) {
		return views.Render(c, listsview.ListsPage(c.Lang, rows, state))
	}
	return views.Render(c, listsview.Lists(c.UserProfile, c.Lang, rows, state))
}

func (m *api) ListNewPage(c *mw.UserContext) error {
	data := listsview.ListFormData{IsNew: true}
	if isHXRequest(c) {
		return views.Render(c, listsview.ListDetailPage(c.Lang, data))
	}
	return views.Render(c, listsview.ListDetail(c.UserProfile, c.Lang, data))
}

func (m *api) ListDetailPage(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	ctx := context.Background()
	l, err := mq.FindMailList(ctx, db.DBob, int32(id))
	if err != nil || l == nil {
		return fmt.Errorf("list not found")
	}

	// Initial: Page 1, keine Filter — datatable übernimmt anschließend.
	rows, total, err := db.QueryMailListRecipientRows(
		c.Request().Context(),
		db.PaginationData{Page: 1, Count: defaultPageSize},
		db.MailListRecipientCriteria{ListID: l.ID},
	)
	if err != nil {
		return err
	}
	state := recipientsTableState(l.ID, db.PagedBindResult[db.MailListRecipientCriteria]{
		Pagination: db.PaginationData{Page: 1, Count: defaultPageSize},
	}, total)

	data := listsview.ListFormData{
		ID:              l.ID,
		Name:            l.Name,
		Description:     l.Description,
		IsNew:           false,
		RecipientsRows:  rows,
		RecipientsState: state,
	}
	if isHXRequest(c) {
		return views.Render(c, listsview.ListDetailPage(c.Lang, data))
	}
	return views.Render(c, listsview.ListDetail(c.UserProfile, c.Lang, data))
}

// ListRecipientsTable beantwortet POST/GET /a/lists/:id/recipients/ —
// gibt NUR die Tabelle zurück (für datatable-Sort/Filter/Page).
func (m *api) ListRecipientsTable(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	bind := db.BindPaginated(c, defaultPageSize, func(b *db.FilterBinder, criteria *db.MailListRecipientCriteria) {
		criteria.ListID = int32(id)
		b.String("email", &criteria.Email)
		b.String("forename", &criteria.Forename)
		b.String("lastname", &criteria.Lastname)
	})
	rows, total, err := db.QueryMailListRecipientRows(c.Request().Context(), bind.Pagination, bind.Criteria)
	if err != nil {
		return err
	}
	state := recipientsTableState(int32(id), bind, total)
	return views.Render(c, listsview.ListRecipientsTable(rows, state))
}

func recipientsTableState(listID int32, bind db.PagedBindResult[db.MailListRecipientCriteria], total int64) datatable.TableState {
	return datatable.TableState{
		Page:      bind.Pagination.Page,
		Count:     bind.Pagination.Count,
		Total:     total,
		SortField: bind.SortField,
		SortDesc:  bind.SortDesc,
		Filters:   bind.Filters,
		Endpoint:  router.Reverse(router.Lists.Recipients, strconv.Itoa(int(listID))),
		Target:    "#list-recipients-table-container",
	}
}

// loadInitialRecipientsPanel rendert das gesamte ListRecipientsPanel mit
// Page=1/keine Filter — wird nach Import und Delete als HTMX-Response
// zurückgegeben (target = #list-recipients-panel).
func loadInitialRecipientsPanel(ctx context.Context, listID int32) ([]*mq.MailListRecipient, datatable.TableState, error) {
	rows, total, err := db.QueryMailListRecipientRows(
		ctx,
		db.PaginationData{Page: 1, Count: defaultPageSize},
		db.MailListRecipientCriteria{ListID: listID},
	)
	if err != nil {
		return nil, datatable.TableState{}, err
	}
	state := recipientsTableState(listID, db.PagedBindResult[db.MailListRecipientCriteria]{
		Pagination: db.PaginationData{Page: 1, Count: defaultPageSize},
	}, total)
	return rows, state, nil
}

// CountListRecipients zählt die Empfänger einer Liste. Wird auf der
// Mailing-Detailseite und in der Mailings-Tabelle aufgerufen, da die Liste
// nach Anlegen des Mailings weiter editiert werden darf — die Zahl wird also
// nicht auf der mailing-Tabelle persistiert.
func CountListRecipients(ctx context.Context, listID int32) (int64, error) {
	return mq.MailListRecipients.Query(
		sm.Where(mq.MailListRecipients.Columns.ListID.EQ(psql.Arg(listID))),
	).Count(ctx, db.DBob)
}

func (m *api) ListCreate(c *mw.UserContext) error {
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			"Name is required",
		)
	}
	description := strings.TrimSpace(c.FormValue("description"))

	ctx := context.Background()
	created, err := mq.MailLists.Insert(&mq.MailListSetter{
		Name:            omit.From(name),
		Description:     omit.From(description),
		CreatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}).One(ctx, db.DBob)
	if err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Lists.Detail, strconv.Itoa(int(created.ID))))
	return c.NoContent(200)
}

func (m *api) ListUpdate(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	ctx := context.Background()
	l, err := mq.FindMailList(ctx, db.DBob, int32(id))
	if err != nil || l == nil {
		return fmt.Errorf("list not found")
	}
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			"Name is required",
		)
	}
	description := strings.TrimSpace(c.FormValue("description"))
	if err := l.Update(ctx, db.DBob, &mq.MailListSetter{
		Name:            omit.From(name),
		Description:     omit.From(description),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	data := listsview.ListFormData{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		IsNew:       false,
	}
	return views.RenderWithToast(c,
		listsview.ListForm(c.Lang, data),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Users.SavedSuccessfully),
	)
}

func (m *api) ListImport(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	ctx := context.Background()
	l, err := mq.FindMailList(ctx, db.DBob, int32(id))
	if err != nil || l == nil {
		return fmt.Errorf("list not found")
	}

	fh, err := c.FormFile("file")
	if err != nil {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Lists.CouldNotReadExcelFile),
		)
	}
	f, err := fh.Open()
	if err != nil {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Lists.CouldNotReadExcelFile),
		)
	}
	defer f.Close()
	bytes, err := io.ReadAll(f)
	if err != nil {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Lists.CouldNotReadExcelFile),
		)
	}

	eL := &importer.ErrorList{}
	imp, err := importer.NewExcelLineImporter[excelRecipientRow](&importer.ExcelLineConfig{
		Config: importer.Config{
			SheetNumber: 1,
			OffsetRow:   1,
			FileBytes:   bytes,
		},
	}, eL)
	if err != nil || eL.HasErrors() {
		msg := i18n.TrLang(c.Lang, i18n.L.Lists.ImportFailed)
		if eL.HasErrors() {
			msg = msg + ": " + eL.Errors[0].Message
		} else if err != nil {
			msg = msg + ": " + err.Error()
		}
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), msg)
	}

	added := 0
	for _, row := range imp.Data {
		email := strings.TrimSpace(row.Email)
		if email == "" || !strings.Contains(email, "@") {
			continue
		}
		_, insErr := mq.MailListRecipients.Insert(&mq.MailListRecipientSetter{
			ListID:   omit.From(l.ID),
			Email:    omit.From(email),
			Forename: omit.From(strings.TrimSpace(row.Forename)),
			Lastname: omit.From(strings.TrimSpace(row.Lastname)),
		}).One(ctx, db.DBob)
		if insErr == nil {
			added++
		}
	}

	rows, state, err := loadInitialRecipientsPanel(ctx, l.ID)
	if err != nil {
		return err
	}
	return views.RenderWithToast(c,
		listsview.ListRecipientsPanel(c.Lang, l.ID, rows, state),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		fmt.Sprintf("%d %s", added, i18n.TrLang(c.Lang, i18n.L.Lists.RecipientsImported)),
	)
}

func (m *api) ListDelete(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	ctx := context.Background()
	l, err := mq.FindMailList(ctx, db.DBob, int32(id))
	if err != nil || l == nil {
		return fmt.Errorf("list not found")
	}
	exists, err := mq.Mailings.Query(
		sm.Where(mq.Mailings.Columns.ListID.EQ(psql.Arg(l.ID))),
	).Exists(ctx, db.DBob)
	if err != nil {
		return err
	}
	if exists {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			"List is in use by a mailing",
		)
	}
	if err := l.Delete(ctx, db.DBob); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Lists.List))
	return c.NoContent(200)
}

// ListRecipientDelete entfernt einen einzelnen Empfänger aus einer Liste.
// Re-rendert das ListRecipientsPanel mit dem aktualisierten Stand.
func (m *api) ListRecipientDelete(c *mw.UserContext) error {
	listID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid list id: %w", err)
	}
	recID, err := strconv.Atoi(c.Param("rec_id"))
	if err != nil {
		return fmt.Errorf("invalid recipient id: %w", err)
	}
	ctx := context.Background()
	l, err := mq.FindMailList(ctx, db.DBob, int32(listID))
	if err != nil || l == nil {
		return fmt.Errorf("list not found")
	}
	r, err := mq.FindMailListRecipient(ctx, db.DBob, int32(recID))
	if err != nil || r == nil || r.ListID != l.ID {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), i18n.TrLang(c.Lang, i18n.L.Lists.RecipientNotFound))
	}
	if err := r.Delete(ctx, db.DBob); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}

	rows, state, err := loadInitialRecipientsPanel(ctx, l.ID)
	if err != nil {
		return err
	}
	return views.RenderWithToast(c,
		listsview.ListRecipientsPanel(c.Lang, l.ID, rows, state),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Lists.RecipientRemoved),
	)
}
