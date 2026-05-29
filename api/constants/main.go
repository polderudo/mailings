package constants

const (
	URIList       = "/list/"
	URIInsert     = "/insert/"
	URIUpsert     = "/upsert/"
	URIDetail     = "/detail/"
	URIDetailStr  = "/detail_str/"
	URIDelete     = "/delete/"
	URIMove       = "/move/"
	URISplit      = "/split/"
	URIAdd        = "/add/"
	URIRemove     = "/remove/"
	URIStart      = "/start/"
	URIStatus     = "/status/"
	URIFinish     = "/finish/"
	URIConfig     = "/config/"
	URINewID      = "/new_id/"
	URIChangesWS  = "/changes_ws/"
	URITest       = "/test/"
	URIFilter     = "/filter/"
	URIFilterP    = "/filter_paged/"
	URISimulate   = "/simulate/"
	URIExport     = "/export/"
	URIExportList = "/export_list/"
	URISearchP    = "/search_paged/"
	URISet        = "/set/"
	URIReset      = "/reset/"

	IDParam = ":id/"
)

type IDDto struct {
	ID int32 `json:"id" query:"id" param:"id" validate:"required"`
}

type ID64Dto struct {
	ID int64 `json:"id" query:"id" param:"id" validate:"required"`
}

type IDStrDto struct {
	ID string `json:"id" query:"id" param:"id" validate:"required"`
}

type PaginationDto struct {
	Limit  int32 `json:"limit" query:"limit" validate:"required"`
	Offset int32 `json:"offset" query:"offset"`
}
