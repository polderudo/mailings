package api

import (
	"app/api/router"
	"app/conf"
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
	"github.com/nakami-lounge-GmbH/ui-components/datatable"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func (m *api) MailingsPage(c *mw.UserContext) error {
	bind := db.BindPaginated(c, 50, func(b *db.FilterBinder, criteria *db.MailingCriteria) {
		b.String("name", &criteria.Name)
		b.String("status", &criteria.Status)
	})
	rows, total, err := db.QueryMailingRows(c.Request().Context(), bind.Pagination, bind.Criteria)
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
		Endpoint:  router.Reverse(router.Mailings.List),
		Target:    "#layout-content-body",
	}
	if isHXRequest(c) {
		return views.Render(c, mailingsview.MailingsListPage(c.Lang, rows, state))
	}
	return views.Render(c, mailingsview.Mailings(c.UserProfile, c.Lang, rows, state))
}

func (m *api) MailingNewPage(c *mw.UserContext) error {
	ctx := context.Background()

	// Sortierung: ID absteigend → die zuletzt angelegten Einträge stehen oben.
	templates, err := mq.MailTemplates.Query(
		sm.OrderBy(mq.MailTemplates.Columns.ID).Desc(),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}
	lists, err := mq.MailLists.Query(
		sm.OrderBy(mq.MailLists.Columns.ID).Desc(),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}
	domains, err := mq.MailDomains.Query(
		sm.Where(mq.MailDomains.Columns.IsActive.EQ(psql.Arg(true))),
		sm.OrderBy(mq.MailDomains.Columns.ID).Desc(),
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
	// Live-Polling während des Versands fragt nur das Info-Panel an (?partial=1).
	if c.QueryParam("partial") == "1" {
		return views.Render(c, mailingsview.MailingInfo(c.Lang, data))
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
		data.Domain = d.Sender
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
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), i18n.TrLang(c.Lang, i18n.L.Mailings.PleasePickASender))
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

	// Synchroner Bulk-POST: gibt schnell die bulk-request-id zurück, das UI zeigt
	// danach direkt status=sending + die Postmark-Daten. Den weiteren Fortschritt
	// zieht der Cron-Poller über GET /email/bulk/{id} nach.
	client := postmark.NewClient(conf.C.Postmark.ServerToken)
	if err := client.SendBulk(ctx, mailing.ID); err != nil {
		slog.Error("postmark send bulk", "err", err, "mailing", mailing.ID)
		mailing, _ = mq.FindMailing(ctx, db.DBob, int32(id))
		data, _ := loadMailingDetail(ctx, mailing)
		return views.RenderWithToast(c,
			mailingsview.MailingInfo(c.Lang, data),
			"error",
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Mailings.MailingFailed),
		)
	}

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
