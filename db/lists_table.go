package db

import (
	"app/mq"
	"context"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// MailListCriteria bündelt die Filter, die die datatable-Page in das Backend
// reicht. Leere Strings = kein Filter.
type MailListCriteria struct {
	Name        string
	Description string
	// Archived: "" oder "active" => nur aktive; "archived" => nur archivierte; "all" => alle.
	Archived string
}

// MailListWithCount ist der View-Typ pro Tabellenzeile: voller MailList-Datensatz
// + abgeleitete Spalte recipient_count (Subselect auf mail_list_recipient).
type MailListWithCount struct {
	*mq.MailList
	RecipientCount int64
}

// scan-Struct für QueryPaginated. `mq.MailList` deckt alle Standardspalten ab,
// recipient_count + _total werden zusätzlich gemappt.
type mailListRowScan struct {
	mq.MailList
	RecipientCount int64 `db:"recipient_count"`
	WithTotal
}

// QueryMailListRows liefert eine paginierte, sortierte Sicht auf die
// Email-Listen inklusive Empfänger-Anzahl pro Liste — alles in einem
// einzigen Round-Trip via COUNT(*) OVER() und Subselect.
func QueryMailListRows(ctx context.Context, p PaginationData, criteria MailListCriteria) ([]MailListWithCount, int64, error) {
	q := SelectAllFrom("mail_list")
	q.Apply(sm.Columns(psql.Raw(`(SELECT COUNT(*) FROM mail_list_recipient r WHERE r.list_id = mail_list.id) AS recipient_count`)))

	if criteria.Name != "" {
		q.Apply(mq.SelectWhere.MailLists.Name.ILike(Like(criteria.Name)))
	}
	if criteria.Description != "" {
		q.Apply(mq.SelectWhere.MailLists.Description.ILike(Like(criteria.Description)))
	}
	switch criteria.Archived {
	case "archived":
		q.Apply(mq.SelectWhere.MailLists.Archived.EQ(true))
	case "all":
		// kein Filter
	default: // "" oder "active"
		q.Apply(mq.SelectWhere.MailLists.Archived.EQ(false))
	}

	return QueryPaginated[mailListRowScan, MailListWithCount](
		ctx, DBob, q, p,
		sm.OrderBy(mq.MailLists.Columns.Name).Asc(),
		func(r mailListRowScan) MailListWithCount {
			ml := r.MailList
			return MailListWithCount{MailList: &ml, RecipientCount: r.RecipientCount}
		},
	)
}
