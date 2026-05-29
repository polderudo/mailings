package views

import (
	"encoding/json"

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
	c.Response().Header().Set("HX-Trigger", string(merged))
}
