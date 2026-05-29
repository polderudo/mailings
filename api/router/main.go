package router

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/labstack/echo/v4"
)

// ERouter ist die Echo-Instanz der laufenden Anwendung.
// Wird in api/main.go nach echo.New() befüllt, damit Reverse() in Views
// die benannten Routen auflösen kann.
var ERouter *echo.Echo

// Reverse löst einen Routennamen (z. B. router.Templates.List) zu seiner URL auf.
// Pfadparameter werden in der Reihenfolge ihres Auftretens an den Pfad gebunden.
//
// Beispiel:
//
//	router.Reverse(router.Templates.Detail, "42")  →  "/a/templates/42/"
func Reverse(name string, pathValues ...string) string {
	if ERouter == nil {
		slog.Error("router.Reverse called before ERouter was set", "name", name)
		return ""
	}
	params := make([]any, len(pathValues))
	for i, v := range pathValues {
		params[i] = v
	}
	return ERouter.Reverse(name, params...)
}

// EnsureAllFieldsSet prüft, dass alle exportierten Felder eines Route-Groups-Struct
// gesetzt sind. Wird via init() pro Route-Group aufgerufen, damit Tippfehler
// in route_names.go beim Start statt zur Laufzeit auffallen.
func EnsureAllFieldsSet[T any](v T) error {
	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return fmt.Errorf("nil pointer")
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct or pointer to struct, got %s", rv.Kind())
	}

	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if !field.CanInterface() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			if field.Len() == 0 {
				return fmt.Errorf("field %s is empty", fieldType.Name)
			}
		case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Func:
			if field.IsNil() {
				return fmt.Errorf("field %s is nil", fieldType.Name)
			}
		default:
			if reflect.ValueOf(field.Interface()).IsZero() {
				return fmt.Errorf("field %s has zero value", fieldType.Name)
			}
		}
	}

	return nil
}
