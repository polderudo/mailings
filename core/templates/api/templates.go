package api

import (
	"app/api/router"
	"app/db"
	"app/i18n"
	"app/mail/format"
	"app/mail/snippet"
	"app/mq"
	"app/mw"
	"app/views"
	templatesview "app/views/pages/main/templates"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/nakami-lounge-GmbH/ui-components/datatable"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func (m *api) TemplatesPage(c *mw.UserContext) error {
	bind := db.BindPaginated(c, 50, func(b *db.FilterBinder, criteria *db.MailTemplateCriteria) {
		b.String("name", &criteria.Name)
		b.String("subject", &criteria.Subject)
		b.String("archived", &criteria.Archived)
	})
	rows, total, err := db.QueryMailTemplateRows(c.Request().Context(), bind.Pagination, bind.Criteria)
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
		Endpoint:  router.Reverse(router.Templates.List),
		Target:    "#layout-content-body",
	}
	if isHXRequest(c) {
		return views.Render(c, templatesview.TemplatesPage(c.Lang, rows, state))
	}
	return views.Render(c, templatesview.Templates(c.UserProfile, c.Lang, rows, state))
}

func (m *api) TemplateNewPage(c *mw.UserContext) error {
	data := templatesview.TemplateFormData{IsNew: true, Format: format.HTML, Snippets: snippet.All()}
	if isHXRequest(c) {
		return views.Render(c, templatesview.TemplateDetailPage(c.Lang, data))
	}
	return views.Render(c, templatesview.TemplateDetail(c.UserProfile, c.Lang, data))
}

func (m *api) TemplateDetailPage(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid template id: %w", err)
	}
	ctx := context.Background()
	t, err := mq.FindMailTemplate(ctx, db.DBob, int32(id))
	if err != nil || t == nil {
		return fmt.Errorf("template not found")
	}
	data := templatesview.TemplateFormData{
		ID:       t.ID,
		Name:     t.Name,
		Subject:  t.Subject,
		Format:   format.Normalize(t.Format),
		BodyHTML: t.BodyHTML,
		BodyText: t.BodyText,
		Archived: t.Archived,
		IsNew:    false,
		Snippets: snippet.All(),
	}
	if isHXRequest(c) {
		return views.Render(c, templatesview.TemplateDetailPage(c.Lang, data))
	}
	return views.Render(c, templatesview.TemplateDetail(c.UserProfile, c.Lang, data))
}

func (m *api) TemplateCreate(c *mw.UserContext) error {
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Templates.TemplateNameIsRequired),
		)
	}
	subject := c.FormValue("subject")
	// Beide Bodies werden immer gespeichert — format entscheidet nur, welcher
	// versendet wird. So verliert ein Formatwechsel den anderen Inhalt nicht.
	bodyHTML := c.FormValue("body_html")
	bodyText := c.FormValue("body_text")
	tplFormat := format.Normalize(c.FormValue("format"))

	ctx := context.Background()
	created, err := mq.MailTemplates.Insert(&mq.MailTemplateSetter{
		Name:            omit.From(name),
		Subject:         omit.From(subject),
		Format:          omit.From(tplFormat),
		BodyHTML:        omit.From(bodyHTML),
		BodyText:        omit.From(bodyText),
		CreatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}).One(ctx, db.DBob)
	if err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Templates.Detail, strconv.Itoa(int(created.ID))))
	return c.NoContent(200)
}

func (m *api) TemplateUpdate(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid template id: %w", err)
	}
	ctx := context.Background()
	t, err := mq.FindMailTemplate(ctx, db.DBob, int32(id))
	if err != nil || t == nil {
		return fmt.Errorf("template not found")
	}
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			i18n.TrLang(c.Lang, i18n.L.Templates.TemplateNameIsRequired),
		)
	}
	subject := c.FormValue("subject")
	bodyHTML := c.FormValue("body_html")
	bodyText := c.FormValue("body_text")
	tplFormat := format.Normalize(c.FormValue("format"))

	if err := t.Update(ctx, db.DBob, &mq.MailTemplateSetter{
		Name:            omit.From(name),
		Subject:         omit.From(subject),
		Format:          omit.From(tplFormat),
		BodyHTML:        omit.From(bodyHTML),
		BodyText:        omit.From(bodyText),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}

	// Aus dem aktualisierten Model rendern (t.Update schreibt die Werte zurück),
	// damit die neu gerenderte Form exakt den gespeicherten Stand zeigt.
	data := templatesview.TemplateFormData{
		ID:       t.ID,
		Name:     t.Name,
		Subject:  t.Subject,
		Format:   format.Normalize(t.Format),
		BodyHTML: t.BodyHTML,
		BodyText: t.BodyText,
		Archived: t.Archived,
		IsNew:    false,
		Snippets: snippet.All(),
	}
	return views.RenderWithToast(c,
		templatesview.TemplateForm(c.Lang, data),
		"success",
		i18n.TrLang(c.Lang, i18n.L.Ui.Success),
		i18n.TrLang(c.Lang, i18n.L.Templates.TemplateSaved),
	)
}

func (m *api) TemplateDelete(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid template id: %w", err)
	}
	ctx := context.Background()

	t, err := mq.FindMailTemplate(ctx, db.DBob, int32(id))
	if err != nil || t == nil {
		return fmt.Errorf("template not found")
	}

	exists, err := mq.Mailings.Query(
		sm.Where(mq.Mailings.Columns.TemplateID.EQ(psql.Arg(t.ID))),
	).Exists(ctx, db.DBob)
	if err != nil {
		return err
	}
	if exists {
		return views.ToastError(c,
			i18n.TrLang(c.Lang, i18n.L.Ui.Error),
			"Template is in use by a mailing",
		)
	}
	if err := t.Delete(ctx, db.DBob); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Templates.List))
	return c.NoContent(200)
}

// TemplateArchive invertiert das archived-Flag des Templates und leitet via
// HX-Redirect zurück auf die Detailseite.
func (m *api) TemplateArchive(c *mw.UserContext) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return fmt.Errorf("invalid template id: %w", err)
	}
	ctx := context.Background()
	t, err := mq.FindMailTemplate(ctx, db.DBob, int32(id))
	if err != nil || t == nil {
		return fmt.Errorf("template not found")
	}
	if err := t.Update(ctx, db.DBob, &mq.MailTemplateSetter{
		Archived:        omit.From(!t.Archived),
		UpdatedByUserID: omitnull.From(c.UserProfile.ID),
		UpdatedAt:       omitnull.From(time.Now()),
	}); err != nil {
		return views.ToastError(c, i18n.TrLang(c.Lang, i18n.L.Ui.Error), err.Error())
	}
	c.Response().Header().Set("HX-Redirect", router.Reverse(router.Templates.Detail, strconv.Itoa(int(t.ID))))
	return c.NoContent(200)
}
