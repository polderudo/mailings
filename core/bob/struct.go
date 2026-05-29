package bob

type TimeZoneDBSlice []*TimeZoneDB
type TimeZoneDB = struct {
	Name   string `db:"name" json:"name"`
	Abbrev string `db:"abbrev" json:"abbrev"`
}
