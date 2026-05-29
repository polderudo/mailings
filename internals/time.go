package internals

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"time"
)

func TimeStr(t time.Time) string {
	return t.Format("02.01.2006 15:04:05")
}

func TimeFileStr(n time.Time) string {
	return fmt.Sprintf("%d.%d.%d__%d.%d.%d", n.Day(), n.Month(), n.Year(), n.Hour(), n.Minute(), n.Second())
}

func TimeStrShortN(t pgtype.Timestamptz) string {
	if t.Valid {
		return t.Time.Format("02.01.06 15:04")
	}
	return ""
}

type MesaTime time.Time

var defaultTimeFormats = []string{
	"02.01.2006 15:04:05",
	"02.01.2006 15:04",
	"02.01.2006",
	"02.01.06 15:04:05",
	"02.01.06 15:04",
	"02.01.06",
	time.RFC3339,
	"2006-01-02",
	"2006-01-02 15:04:05",
	"2006-01-02 03:04:05 PM",
	"01/02/2006",
	"01/02/2006 15:04:05",
	"01/02/2006 03:04:05 PM",
}

func (ct *MesaTime) UnmarshalParam(src string) error {
	for _, format := range defaultTimeFormats {
		t, err := time.Parse(format, src)
		if err == nil {
			*ct = MesaTime(t)
			return nil
		}
	}
	return fmt.Errorf("unable to parse time string '%s' with any of the provided formats", src)
}

func (mt *MesaTime) UnmarshalJSON(data []byte) error {
	// data contains the raw JSON value as a byte slice.
	// For strings, it will include the surrounding quotes.

	var s string
	// Unmarshal the raw JSON data into a string first
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("MesaTime: expected a string, got %s", data)
	}

	var parsedTime time.Time
	var err error

	for _, format := range defaultTimeFormats {
		parsedTime, err = time.Parse(format, s)
		if err == nil {
			*mt = MesaTime(parsedTime) // Assign the parsed time to the receiver
			return nil
		}
	}

	return fmt.Errorf("MesaTime: unable to parse time string '%s' with any of the supported formats", s)
}
