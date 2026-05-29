package db

import (
	"time"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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

// ApplySortsLimitOffset applies all the sorting and limits to the query
func ApplySortsLimitOffset(q bob.BaseQuery[*dialect.SelectQuery], p PaginationData, defaultSort dialect.OrderBy[*dialect.SelectQuery]) { //ApplySortsLimitOffset[T any, Ts ~[]T](q *psql.ViewQuery[T, Ts], p PaginationData) {
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
