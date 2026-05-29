package api

import (
	"app/api/router"
	listsapi "app/core/lists/api"
	"app/db"
	"app/i18n"
	"app/mail/postmark"
	"app/mq"
	"app/mw"
	"app/views"
	mailingsview "app/views/pages/main/mailings"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	globaltable "github.com/nakami-lounge-GmbH/ui-components/table"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func (m *api) MailingsPage(c *mw.UserContext) error {
	request := globaltable.ReadRequest(c.Request(), mailingsview.CurrentMailingsTableID)
	sorts := make(db.SortParams, len(request.Pagination.Sorts))
	for i, s := range request.Pagination.Sorts {
		sorts[i] = db.SortParam{Field: s.Field, IsDesc: s.IsDesc}
	}
	rows, total, err := db.LoadMailingTableRows(c.Request().Context(), db.FilterRequest[map[string]string]{
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
	if isHXRequest(c) {
		if c.Request().Header.Get("HX-Target") == mailingsview.CurrentMailingsTableID {
			return views.Render(c, mailingsview.CurrentMailingsTable(c.Lang, rows, request, total))
		}
		return views.Render(c, mailingsview.MailingsListPage(c.Lang, rows, request, total))
	}
	return views.Render(c, mailingsview.Mailings(c.UserProfile, c.Lang, rows, request, total))
}

func (m *api) MailingNewPage(c *mw.UserContext) error {
	ctx := context.Background()

	templates, err := mq.MailTemplates.Query(
		sm.OrderBy(mq.MailTemplates.Columns.Name),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}
	lists, err := mq.MailLists.Query(
		sm.OrderBy(mq.MailLists.Columns.Name),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}
	domains, err := mq.MailDomains.Query(
		sm.Where(mq.MailDomains.Columns.IsActive.EQ(psql.Arg(true))),
		sm.OrderBy(mq.MailDomains.Columns.Domain),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}
	data := mailingsview.MailingNewFormData{
		Templates: templates,
		Lists:     lists,
		Domains:   domains,
	}
	if isHXRequest(c) {
		return views.Render(c, mailingsview.MailingNewPage(c.Lang, data))
	}
	return views.Render(c, mailingsview.MailingNew(c.UserProfile, c.Lang, data))
}

func (m *api) MailingDetailPage(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid mailing id: %w", err)
	}
	ctx := context.Background()
	mailing, err := mq.FindMailing(ctx, db.DBob, int32(id))
	if err != nil || mailing == nil {
		return fmt.Errorf("mailing not found")
	}
	data, err := loadMailingDetail(ctx, mailing)
	if err != nil {
		return err
	}
	if isHXRequest(c) {
		return views.Render(c, mailingsview.MailingDetailPage(c.Lang, data))
	}
	return views.Render(c, mailingsview.MailingDetail(c.UserProfile, c.Lang, data))
}

func loadMailingDetail(ctx context.Context, mailing *mq.Mailing) (mailingsview.MailingDetailData, error) {
	data := mailingsview.MailingDetailData{Mailing: mailing}
	if t, err := mq.FindMailTemplate(ctx, db.DBob, mailing.TemplateID); err == nil && t != nil {
		data.TemplateName = t.Name
	}
	if l, err := mq.FindMailList(ctx, db.DBob, mailing.ListID); err == nil && l != nil {
		data.ListName = l.Name
	}
	if d, err := mq.FindMailDomain(ctx, db.DBob, mailing.DomainID); err == nil && d != nil {
		data.Domain = d.Domain
	}
	// Recipient-Count wird immer live aus mail_list_recipient ermittelt — die
	// Liste darf sich nach Anlegen des Mailings noch ändern, daher steht die
	// Zahl nicht auf der mailing-Tabelle.
	if cnt, err := listsapi.CountListRecipients(ctx, mailing.ListID); err == nil {
		data.RecipientCount = cnt
	}
	return data, nil
}

func (m *api) MailingCreate(c *mw.UserContext) error {
	name := strings.TrimSpace(c.FormValue("name"))
	templateID, err := strconv.Atoi(c.FormValue("template_id"))
	if err != nil || templateID == 0 {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), i18n.TrLang(c.Lang, i18n.L.Mailings.PleasePickATemplate))
	}
	listID, err := strconv.Atoi(c.FormValue("list_id"))
	if err != nil || listID == 0 {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), i18n.TrLang(c.Lang, i18n.L.Mailings.PleasePickAnEmailList))
	}
	domainID, err := strconv.Atoi(c.FormValue("domain_id"))
	if err != nil || domainID == 0 {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), i18n.TrLang(c.Lang, i18n.L.Mailings.PleasePickADomain))
	}

	ctx := context.Background()
	tpl, err := mq.FindMailTemplate(ctx, db.DBob, int32(templateID))
	if err != nil || tpl == nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), "Template not found")
	}

	created, err := mq.Mailings.Insert(&mq.MailingSetter{
		Name:            omit.From(name),
		TemplateID:      omit.From(int32(templateID)),
		ListID:          omit.From(int32(listID)),
		DomainID:        omit.From(int32(domainID)),
		Status:          omit.From("draft"),
		SubjectSnapshot: omit.From(tpl.Subject),
		BodySnapshot:    omit.From(tpl.BodyHTML),
		CreatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}).One(ctx, db.DBob)
	if err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}

	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Mailings.Detail, strconv.Itoa(int(created.ID))))
	return c.NoContent(200)
}

func (m *api) MailingStart(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid mailing id: %w", err)
	}
	ctx := context.Background()
	mailing, err := mq.FindMailing(ctx, db.DBob, int32(id))
	if err != nil || mailing == nil {
		return fmt.Errorf("mailing not found")
	}
	if mailing.Status != "draft" {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), "Mailing not in draft status")
	}

	d, err := mq.FindMailDomain(ctx, db.DBob, mailing.DomainID)
	if err != nil || d == nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), "Domain not found")
	}

	// TODO: postmark server token aus conf.C.Postmark.ServerToken laden
	client := postmark.NewClient("")
	go func() {
		if err := client.SendBroadcast(context.Background(), mailing.ID); err != nil {
			slog.Error("send broadcast", "err", err, "mailing", mailing.ID)
		}
	}()

	mailing, _ = mq.FindMailing(ctx, db.DBob, int32(id))
	data, err := loadMailingDetail(ctx, mailing)
	if err != nil {
		return err
	}
	return views.RenderWithToast(c,
		mailingsview.MailingInfo(c.Lang, data),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Mailings.SendingStarted),
	)
}
