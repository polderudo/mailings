package db

import (
	"app/mq"
	"context"

	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// MailBlacklistCriteria bündelt die Filter für die Blacklist-Tabelle.
type MailBlacklistCriteria struct {
	Email  string
	Reason string
}

type mailBlacklistRowScan struct {
	mq.MailBlacklist
	WithTotal
}

// QueryMailBlacklistRows liefert eine paginierte Sicht auf die globale Sperrliste.
func QueryMailBlacklistRows(ctx context.Context, p PaginationData, criteria MailBlacklistCriteria) ([]*mq.MailBlacklist, int64, error) {
	q := SelectAllFrom("mail_blacklist")

	if criteria.Email != "" {
		q.Apply(mq.SelectWhere.MailBlacklists.Email.ILike(Like(criteria.Email)))
	}
	if criteria.Reason != "" {
		q.Apply(mq.SelectWhere.MailBlacklists.Reason.ILike(Like(criteria.Reason)))
	}

	return QueryPaginated[mailBlacklistRowScan, *mq.MailBlacklist](
		ctx, DBob, q, p,
		sm.OrderBy(mq.MailBlacklists.Columns.CreatedAt).Desc(),
		func(r mailBlacklistRowScan) *mq.MailBlacklist {
			mb := r.MailBlacklist
			return &mb
		},
	)
}
