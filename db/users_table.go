package db

import (
	"app/mq"
	"context"
	"strings"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// UserProfileCriteria bündelt die Filter für die User-Tabelle.
//   - FullName matcht gegen forename, lastname und email (ILIKE).
//   - Status filtert auf "active" / "disabled" — alles andere ist kein Filter.
type UserProfileCriteria struct {
	FullName string
	Status   string
}

type userProfileRowScan struct {
	mq.UserProfile
	WithTotal
}

// QueryUserProfileRows liefert eine paginierte Sicht auf alle Benutzer.
func QueryUserProfileRows(ctx context.Context, p PaginationData, criteria UserProfileCriteria) ([]*mq.UserProfile, int64, error) {
	q := SelectAllFrom("user_profile")

	if criteria.FullName != "" {
		lk := Like(criteria.FullName)
		q.Apply(sm.Where(psql.Raw("(CONCAT_WS(' ', forename, lastname) ILIKE ? OR email ILIKE ?)", lk, lk)))
	}
	switch strings.ToLower(strings.TrimSpace(criteria.Status)) {
	case "active":
		q.Apply(mq.SelectWhere.UserProfiles.Disabled.EQ(false))
	case "disabled":
		q.Apply(mq.SelectWhere.UserProfiles.Disabled.EQ(true))
	}

	return QueryPaginated[userProfileRowScan, *mq.UserProfile](
		ctx, DBob, q, p,
		sm.OrderBy(mq.UserProfiles.Columns.Forename).Asc(),
		func(r userProfileRowScan) *mq.UserProfile {
			up := r.UserProfile
			return &up
		},
	)
}
