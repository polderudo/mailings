package db

import (
	"app/mq"
	"context"

	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// MailTemplateCriteria bündelt die Filter für die Template-Tabelle.
type MailTemplateCriteria struct {
	Name    string
	Subject string
}

type mailTemplateRowScan struct {
	mq.MailTemplate
	WithTotal
}

// QueryMailTemplateRows liefert eine paginierte Sicht auf die HTML-Templates.
func QueryMailTemplateRows(ctx context.Context, p PaginationData, criteria MailTemplateCriteria) ([]*mq.MailTemplate, int64, error) {
	q := SelectAllFrom("mail_template")

	if criteria.Name != "" {
		q.Apply(mq.SelectWhere.MailTemplates.Name.ILike(Like(criteria.Name)))
	}
	if criteria.Subject != "" {
		q.Apply(mq.SelectWhere.MailTemplates.Subject.ILike(Like(criteria.Subject)))
	}

	return QueryPaginated[mailTemplateRowScan, *mq.MailTemplate](
		ctx, DBob, q, p,
		sm.OrderBy(mq.MailTemplates.Columns.Name).Asc(),
		func(r mailTemplateRowScan) *mq.MailTemplate {
			mt := r.MailTemplate
			return &mt
		},
	)
}
