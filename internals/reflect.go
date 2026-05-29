package internals

import (
	"reflect"
	"strings"
)

// JsonFieldName returns the json tag for the given fieldName
func JsonFieldName[T any](v *T, fieldName string) (name string) {
	t := reflect.TypeOf(*v)

	f, found := t.FieldByName(fieldName)
	if found {
		tag := f.Tag.Get("json")
		if tag == "" {
			return f.Name
		}

		if tag == "-" {
			return ""
		}

		if i := strings.Index(tag, ","); i != -1 {
			if i == 0 {
				return f.Name
			} else {
				return tag[:i]
			}
		}
		return tag
	}

	return fieldName
}
