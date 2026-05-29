package db

import (
	"context"
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// TimeFilter defines a filter operation for time data
// operator might be "<", ">", "<=", ">=", "between", and "="
// The filter automatically uses 00:00:00 and/or 23:59:59 for doing the filtering if withTime is false
// if withTime==false, and operator == "=" we use time between with 00:00 to 23:59. For "<" ">"  accrdingly
type TimeFilter struct {
	Operator   string     `json:"operator"` //might be "<", ">", "<=", ">=", "between", and "="
	TimeValue1 time.Time  `json:"time_value_1"`
	TimeValue2 *time.Time `json:"time_value_2,omitempty"`
	WithTime   bool       `json:"with_time"`
}

func NewTimeFilter(tf *TimeFilter) *TimeFilter {
	if tf == nil {
		return nil
	}

	//we assume we get UTC times, so we have to convert them to local, because the DB stored the time with timezone.
	tz := time.Local
	tf.TimeValue1 = tf.TimeValue1.In(tz)
	if tf.TimeValue2 != nil {
		tt := tf.TimeValue2.In(tz)
		tf.TimeValue2 = &tt
	}

	if !tf.WithTime {
		switch tf.Operator {
		case OpEq:
			tf.TimeValue1 = time.Date(tf.TimeValue1.Year(), tf.TimeValue1.Month(), tf.TimeValue1.Day(), 0, 0, 0, 0, tz)
			t2 := time.Date(tf.TimeValue1.Year(), tf.TimeValue1.Month(), tf.TimeValue1.Day(), 23, 59, 59, 0, tz)
			tf.TimeValue2 = &t2
		case OpLT, OpGTE:
			tf.TimeValue1 = time.Date(tf.TimeValue1.Year(), tf.TimeValue1.Month(), tf.TimeValue1.Day(), 0, 0, 0, 0, tz)
		case OpGT, OpLTE:
			tf.TimeValue1 = time.Date(tf.TimeValue1.Year(), tf.TimeValue1.Month(), tf.TimeValue1.Day(), 23, 59, 59, 0, tz)
		case OpBetween:
			tf.TimeValue1 = time.Date(tf.TimeValue1.Year(), tf.TimeValue1.Month(), tf.TimeValue1.Day(), 0, 0, 0, 0, tz)
			t2 := time.Date(tf.TimeValue2.Year(), tf.TimeValue2.Month(), tf.TimeValue2.Day(), 23, 59, 59, 0, tz)
			tf.TimeValue2 = &t2
		}
	}

	return tf
}

const (
	OpEq      = "="
	OpBetween = "between"
	OpLT      = "<"
	OpLTE     = "<="
	OpGT      = ">"
	OpGTE     = ">="
)

// SelectAllFrom returns a query that selects all columns from the given table
// (SELECT table.* FROM table). Use with ApplySortsLimitOffset and bob.All.
func SelectAllFrom(table string) bob.BaseQuery[*dialect.SelectQuery] {
	return psql.Select(
		sm.From(table),
		sm.Columns(table+".*"),
	)
}

// QueryPaginated applies sorting/limit/offset, executes the query in one DB
// round-trip (including COUNT(*) OVER() AS _total), and extracts the results.
//
// Row is the local scan struct (embedding the model and db.WithTotal).
// T is the model pointer type returned to the caller.
// fn extracts T from Row; Go infers both type parameters from fn.
//
// Example:
//
//	rawData, total, err := db.QueryPaginated(
//	    ctx, db.DBob, q, r.Pagination,
//	    sm.OrderBy(cn.MatDefID.Name).Asc(),
//	    func(r matDefRow) *mq.MatDef { return &r.MatDef },
//	)
func QueryPaginated[Row interface{ GetTotal() int64 }, T any](
	ctx context.Context,
	exec bob.Executor,
	q bob.BaseQuery[*dialect.SelectQuery],
	p PaginationData,
	defaultSort dialect.OrderBy[*dialect.SelectQuery],
	fn func(Row) T,
) ([]T, int64, error) {
	ApplySortsLimitOffset(q, p, defaultSort)
	rows, err := bob.All(ctx, exec, q, scan.StructMapper[Row]())
	if err != nil {
		return nil, 0, err
	}
	data, total := extractRows(rows, fn)
	return data, total, nil
}

// ApplySortsLimitOffset applies sorting, limit, and offset to the query.
// It also appends COUNT(*) OVER() AS _total to the SELECT list so that
// callers can retrieve the pre-pagination total in a single DB round-trip.
// Existing callers that scan into the plain model type are unaffected because
// scan.StructMapper ignores unknown columns.
func ApplySortsLimitOffset(q bob.BaseQuery[*dialect.SelectQuery], p PaginationData, defaultSort dialect.OrderBy[*dialect.SelectQuery]) {
	q.Apply(sm.Columns(psql.Raw("COUNT(*) OVER() AS _total")))
	ApplySortsToQuery(q, p.Sorts, defaultSort)
	applyLimit(q, p.Count)
	applyOffset(q, p.Offset())
}

func ApplySortsToQuery(q bob.BaseQuery[*dialect.SelectQuery], s SortParams, defaultSort dialect.OrderBy[*dialect.SelectQuery]) {
	if len(s) == 0 {
		q.Apply(defaultSort)
	} else {
		for _, v := range s {
			ss := sm.OrderBy(psql.Quote(v.Field))
			if v.IsDesc {
				ss = ss.Desc()
			} else {
				ss = ss.Asc()
			}
			q.Apply(ss)
		}
	}
}

func applyLimit(q bob.BaseQuery[*dialect.SelectQuery], limit int64) {
	if limit > 0 {
		q.Apply(sm.Limit(limit))
	}
}

func applyOffset(q bob.BaseQuery[*dialect.SelectQuery], offset int64) {
	if offset > 1 {
		q.Apply(sm.Offset(offset))
	}
}

func ApplyTimeFilter[T any, Ts ~[]T](q *psql.ViewQuery[T, Ts], tf *TimeFilter, wm psql.WhereMod[*dialect.SelectQuery, time.Time]) {
	switch tf.Operator {
	case OpEq, OpBetween:
		q.Apply(psql.WhereAnd(wm.GTE(tf.TimeValue1), wm.LTE(*tf.TimeValue2)))
	case OpLT:
		q.Apply(wm.LT(tf.TimeValue1))
	case OpLTE:
		q.Apply(wm.LTE(tf.TimeValue1))
	case OpGT:
		q.Apply(wm.GT(tf.TimeValue1))
	case OpGTE:
		q.Apply(wm.GTE(tf.TimeValue1))
	}
}

func ApplyTimeFilterNull[T any, Ts ~[]T](q *psql.ViewQuery[T, Ts], tf *TimeFilter, wm psql.WhereNullMod[*dialect.SelectQuery, time.Time]) {
	ApplyTimeFilter(q, tf, wm.WhereMod)
}
