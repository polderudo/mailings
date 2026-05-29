package db

import (
	"app/mq"
	"context"

	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// MailListRecipientCriteria bündelt die Filter für die Empfänger-Tabelle.
type MailListRecipientCriteria struct {
	ListID   int32 // PFLICHT
	Email    string
	Forename string
	Lastname string
}

type mailListRecipientRowScan struct {
	mq.MailListRecipient
	WithTotal
}

// QueryMailListRecipientRows liefert eine paginierte, sortierte Sicht auf die
// Empfänger einer bestimmten Email-Liste.
func QueryMailListRecipientRows(ctx context.Context, p PaginationData, criteria MailListRecipientCriteria) ([]*mq.MailListRecipient, int64, error) {
	q := SelectAllFrom("mail_list_recipient")
	q.Apply(mq.SelectWhere.MailListRecipients.ListID.EQ(criteria.ListID))

	if criteria.Email != "" {
		q.Apply(mq.SelectWhere.MailListRecipients.Email.ILike(Like(criteria.Email)))
	}
	if criteria.Forename != "" {
		q.Apply(mq.SelectWhere.MailListRecipients.Forename.ILike(Like(criteria.Forename)))
	}
	if criteria.Lastname != "" {
		q.Apply(mq.SelectWhere.MailListRecipients.Lastname.ILike(Like(criteria.Lastname)))
	}

	return QueryPaginated[mailListRecipientRowScan, *mq.MailListRecipient](
		ctx, DBob, q, p,
		sm.OrderBy(mq.MailListRecipients.Columns.Email).Asc(),
		func(r mailListRecipientRowScan) *mq.MailListRecipient {
			mr := r.MailListRecipient
			return &mr
		},
	)
}
