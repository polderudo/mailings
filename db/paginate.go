// Übernommen aus mesa2/core/db/paginate.go (read-only Referenz).
// Stellt das BindPaginated-Pattern für datatable-basierte HTMX-Seiten bereit.
package db

import (
	"strconv"
	"time"
)

// FormGetter is satisfied by *echo.Context and any wrapper that provides
// form/query value access. It is intentionally narrow so the db package
// does not need to import the echo package.
type FormGetter interface {
	FormValue(string) string
}

// FilterBinder is passed to the caller's closure so it can register each
// filterable field. It reads the field value from the HTTP request and
// accumulates non-empty values into the Filters map.
type FilterBinder struct {
	g       FormGetter
	filters map[string]string
}

// String binds a single string filter field from the request.
func (b *FilterBinder) String(field string, target *string) {
	val := b.g.FormValue(field)
	*target = val
	if val != "" {
		b.filters[field] = val
	}
}

// BoolPtr binds a tri-state boolean filter from the request.
// Empty string = no filter (target remains nil); "true" = *true; "false" = *false.
func (b *FilterBinder) BoolPtr(field string, target **bool) {
	val := b.g.FormValue(field)
	switch val {
	case "true":
		v := true
		*target = &v
		b.filters[field] = val
	case "false":
		v := false
		*target = &v
		b.filters[field] = val
	}
}

// TimeFilter binds three form fields produced by the DateInput component
// ({field}_op, {field}_date1, {field}_date2) into a *TimeFilter.
// Values are stored in Filters under the same keys for UI pre-fill.
func (b *FilterBinder) TimeFilter(field string, target **TimeFilter) {
	op := b.g.FormValue(field + "_op")
	if op == "" {
		return
	}
	date1Str := b.g.FormValue(field + "_date1")
	d1, err := time.Parse("2006-01-02", date1Str)
	if err != nil {
		return
	}
	tf := &TimeFilter{Operator: op, TimeValue1: d1}
	date2Str := ""
	if op == OpBetween {
		date2Str = b.g.FormValue(field + "_date2")
		d2, err := time.Parse("2006-01-02", date2Str)
		if err != nil {
			return
		}
		tf.TimeValue2 = &d2
	}
	*target = tf
	b.filters[field+"_op"] = op
	b.filters[field+"_date1"] = date1Str
	if date2Str != "" {
		b.filters[field+"_date2"] = date2Str
	}
}

// TimeFilterValue holds the parsed form values for a date filter field,
// ready to pass directly to the DateInput component.
type TimeFilterValue struct {
	Name  string // form field name
	Op    string // operator (=, >, <, >=, <=, between)
	Date1 string // YYYY-MM-DD
	Date2 string // YYYY-MM-DD (between only)
}

// TimeFilterFormValues reads the three filter map entries written by TimeFilter
// ({name}_op, {name}_date1, {name}_date2) into a TimeFilterValue for UI pre-fill.
func TimeFilterFormValues(name string, filters map[string]string) TimeFilterValue {
	return TimeFilterValue{
		Name:  name,
		Op:    filters[name+"_op"],
		Date1: filters[name+"_date1"],
		Date2: filters[name+"_date2"],
	}
}

// PagedBindResult holds everything ListHTML (or any paginated HTML handler)
// needs after calling BindPaginated.
//
// Pagination holds the db-layer pagination state ready to pass to a query
// function. Filters is the subset of non-empty filter values ready to pass
// to datatable.TableState.
type PagedBindResult[T any] struct {
	Pagination PaginationData
	SortField  string
	SortDesc   bool
	Criteria   T
	Filters    map[string]string
}

// WithTotal is embedded in per-handler scan structs to capture the window
// function total injected by ApplySortsLimitOffset.
//
//	type matDefRow struct {
//	    mq.MatDef
//	    db.WithTotal
//	}
type WithTotal struct {
	Total int64 `db:"_total"`
}

// GetTotal satisfies the constraint used by extractRows.
func (w WithTotal) GetTotal() int64 { return w.Total }

// extractRows converts a slice of scan rows (each embedding db.WithTotal) into
// a slice of model values and the pre-pagination total count.
func extractRows[Row interface{ GetTotal() int64 }, T any](rows []Row, fn func(Row) T) ([]T, int64) {
	var total int64
	result := make([]T, len(rows))
	for i, row := range rows {
		if i == 0 {
			total = row.GetTotal()
		}
		result[i] = fn(row)
	}
	return result, total
}

// BindPaginated parses the standard pagination query parameters (sort, desc,
// page, count) from the HTTP request and then invokes fill so the caller can
// bind its own filter fields via the FilterBinder.
//
// It works for both GET requests (values come from the URL query string) and
// POST requests (values come from the form body), because net/http's
// FormValue reads both sources.
//
// defaultCount is used when the request carries no count parameter or count ≤ 0.
func BindPaginated[T any](g FormGetter, defaultCount int64, fill func(*FilterBinder, *T)) PagedBindResult[T] {
	sortField := g.FormValue("sort")
	sortDesc := g.FormValue("desc") == "true"

	page, _ := strconv.Atoi(g.FormValue("page"))
	if page < 1 {
		page = 1
	}

	count, _ := strconv.ParseInt(g.FormValue("count"), 10, 64)
	if count <= 0 {
		count = defaultCount
	}

	pd := PaginationData{Page: page, Count: count}
	if sortField != "" {
		pd.Sorts = SortParams{{Field: sortField, IsDesc: sortDesc}}
	}

	b := &FilterBinder{g: g, filters: make(map[string]string)}
	var criteria T
	fill(b, &criteria)

	return PagedBindResult[T]{
		Pagination: pd,
		SortField:  sortField,
		SortDesc:   sortDesc,
		Criteria:   criteria,
		Filters:    b.filters,
	}
}
