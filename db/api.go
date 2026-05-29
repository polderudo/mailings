package db

type FilterRequest[T any] struct {
	P              PaginationData `json:"pagination"`
	FilterCriteria T              `json:"filter_criteria"`
}

type FilterResponse[T any] struct {
	PaginationResponse
	Data T `json:"data"`
}
