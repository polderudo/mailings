package views

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

type toastPayload struct {
	Variant string `json:"variant"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func RenderWithToast(c echo.Context, comp templ.Component, variant, title, msg string) error {
	setToastHeader(c, variant, title, msg)
	return Render(c, comp)
}

func ToastSuccess(c echo.Context, title, msg string) error {
	setToastHeader(c, "success", title, msg)
	c.Response().Header().Set("HX-Reswap", "none")
	return c.NoContent(200)
}

func ToastError(c echo.Context, title, msg string) error {
	setToastHeader(c, "error", title, msg)
	c.Response().Header().Set("HX-Reswap", "none")
	return c.NoContent(200)
}

func setToastHeader(c echo.Context, variant, title, msg string) {
	data, _ := json.Marshal(toastPayload{Variant: variant, Title: title, Message: msg})

	existing := c.Response().Header().Get("HX-Trigger")
	var triggers map[string]json.RawMessage
	if existing != "" {
		if err := json.Unmarshal([]byte(existing), &triggers); err != nil {
			triggers = map[string]json.RawMessage{existing: json.RawMessage("null")}
		}
	} else {
		triggers = make(map[string]json.RawMessage)
	}
	triggers["vkb:toast"] = json.RawMessage(data)

	merged, _ := json.Marshal(triggers)
	c.Response().Header().Set("HX-Trigger", string(asciiEscapeJSON(merged)))
}

// asciiEscapeJSON wandelt alle Non-ASCII-Runen im JSON-Bytestrom in `\uXXXX`-
// Escapes um. HTTP-Header sind nicht UTF-8-sicher (Browser interpretieren sie
// nach RFC 7230 als ISO-8859-1), daher würden Umlaute & Co. sonst in
// HX-Trigger-Payloads zu Mojibake werden. Da `\uXXXX` valides JSON ist, kommt
// das ursprüngliche Zeichen nach `JSON.parse` im Browser zurück.
func asciiEscapeJSON(b []byte) []byte {
	if !needsASCIIEscape(b) {
		return b
	}
	var out bytes.Buffer
	out.Grow(len(b))
	s := string(b)
	for _, r := range s {
		if r < 0x80 {
			out.WriteRune(r)
			continue
		}
		if r <= 0xFFFF {
			fmt.Fprintf(&out, `\u%04x`, r)
			continue
		}
		// Encode supplementary planes as a UTF-16 surrogate pair.
		r -= 0x10000
		hi := 0xD800 + (r >> 10)
		lo := 0xDC00 + (r & 0x3FF)
		fmt.Fprintf(&out, `\u%04x\u%04x`, hi, lo)
	}
	return out.Bytes()
}

func needsASCIIEscape(b []byte) bool {
	for _, c := range b {
		if c >= 0x80 {
			return true
		}
	}
	return false
}
