package db

import (
	"fmt"
)

type SortParam struct {
	Field  string `json:"field"`
	IsDesc bool   `json:"is_desc"`
}
type SortParams []SortParam

type PaginationResponse struct {
	Total        int64    `json:"total"`
	FilterFields []string `json:"filter_fields"`
	SortFields   []string `json:"sort_fields"`
}

type FilterParam struct {
	Value interface{} `json:"value"`
}
type FilterParams map[string]FilterParam

func (m *FilterParams) Has(f string) bool {
	_, ok := (*m)[f]
	return ok
}

type PaginationData struct {
	Sorts   SortParams   `schema:"sorts" json:"sorts"`
	Filters FilterParams `schema:"filters" json:"filters"`
	Page    int          `schema:"page" json:"page"`
	Count   int64        `schema:"count" json:"count"`
}

func (p *PaginationData) Offset() int64 {
	var offset int64
	if p.Page > 1 && p.Count > 0 {
		offset = int64(p.Page-1) * p.Count
	}
	return offset
}

func (pr *PaginationResponse) SetFilterAndSort(fields ...string) {
	pr.FilterFields = append(pr.FilterFields, fields...)
	pr.SortFields = append(pr.SortFields, fields...)
}

func (pr *PaginationResponse) AddSort(fields ...string) {
	pr.SortFields = append(pr.SortFields, fields...)
}

func (pr *PaginationResponse) AddFilter(fields ...string) {
	pr.FilterFields = append(pr.FilterFields, fields...)
}

func (s SortParams) IsAsc(field string) bool {
	for _, ss := range s {
		if ss.Field == field {
			return !ss.IsDesc
		}
	}
	return false
}

func (s SortParams) IsDesc(field string) bool {
	for _, ss := range s {
		if ss.Field == field {
			return ss.IsDesc
		}
	}
	return false
}

type Sort struct {
	IsAsc  bool
	IsDesc bool
}

func (s SortParams) GetSorts(field string) Sort {
	var ret Sort
	for _, ss := range s {
		if ss.Field == field {
			ret.IsAsc = !ss.IsDesc
			ret.IsDesc = ss.IsDesc
		}
	}
	return ret
}

type FieldAdd struct {
	Field     string
	AddFilter bool
	AddSort   bool
}

func (pr *PaginationResponse) AddFields(fields []FieldAdd) {
	for _, f := range fields {
		if f.AddFilter {
			pr.FilterFields = append(pr.FilterFields, f.Field)
		}
		if f.AddSort {
			pr.SortFields = append(pr.SortFields, f.Field)
		}
	}
}

func Like(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%%%v%%", v)
}
