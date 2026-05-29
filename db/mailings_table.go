package db

import (
	"app/mq"
	"context"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// MailingCriteria bündelt die Filter für die Mailings-Tabelle.
type MailingCriteria struct {
	Name   string
	Status string
}

// MailingWithMeta ist die View-Typ-Repräsentation eines Mailings inkl.
// joined Template/List/Domain-Namen und live ermitteltem Recipient-Count.
type MailingWithMeta struct {
	*mq.Mailing
	TemplateName   string
	ListName       string
	DomainName     string
	RecipientCount int64
}

type mailingRowScan struct {
	mq.Mailing
	TemplateName   string `db:"template_name"`
	ListName       string `db:"list_name"`
	DomainName     string `db:"domain_name"`
	RecipientCount int64  `db:"recipient_count"`
	WithTotal
}

// QueryMailingRows liefert die paginierte Sicht mit Template-/Listen-/Domain-Namen
// und live ermittelter Empfänger-Anzahl.
func QueryMailingRows(ctx context.Context, p PaginationData, criteria MailingCriteria) ([]MailingWithMeta, int64, error) {
	q := SelectAllFrom("mailing")
	q.Apply(sm.Columns(
		psql.Raw("mail_template.name AS template_name"),
		psql.Raw("mail_list.name AS list_name"),
		psql.Raw("mail_domain.sender AS domain_name"),
		psql.Raw("(SELECT COUNT(*) FROM mail_list_recipient r WHERE r.list_id = mailing.list_id) AS recipient_count"),
	))
	q.Apply(sm.InnerJoin("mail_template ON mail_template.id = mailing.template_id"))
	q.Apply(sm.InnerJoin("mail_list ON mail_list.id = mailing.list_id"))
	q.Apply(sm.InnerJoin("mail_domain ON mail_domain.id = mailing.domain_id"))

	if criteria.Name != "" {
		q.Apply(mq.SelectWhere.Mailings.Name.ILike(Like(criteria.Name)))
	}
	if criteria.Status != "" {
		q.Apply(mq.SelectWhere.Mailings.Status.ILike(Like(criteria.Status)))
	}

	return QueryPaginated[mailingRowScan, MailingWithMeta](
		ctx, DBob, q, p,
		sm.OrderBy(mq.Mailings.Columns.CreatedAt).Desc(),
		func(r mailingRowScan) MailingWithMeta {
			m := r.Mailing
			return MailingWithMeta{
				Mailing:        &m,
				TemplateName:   r.TemplateName,
				ListName:       r.ListName,
				DomainName:     r.DomainName,
				RecipientCount: r.RecipientCount,
			}
		},
	)
}
