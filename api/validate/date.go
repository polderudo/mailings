package validate

import (
	"reflect"
	"time"
)

// normalizeTimesToUTC walks through structs/slices/pointers and converts any time.Time / *time.Time to UTC.
func normalizeTimesToUTC(v any) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return
	}
	normalizeValueToUTC(rv)
}

var timeType = reflect.TypeOf(time.Time{})

func normalizeValueToUTC(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	// Unwrap interfaces
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		normalizeValueToUTC(v.Elem())
		return
	}

	// Unwrap pointers
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		normalizeValueToUTC(v.Elem())
		return
	}

	// Convert direct time.Time
	if v.Type() == timeType {
		if !v.CanSet() {
			return
		}
		t := v.Interface().(time.Time)
		if t.IsZero() {
			return
		}
		//v.Set(reflect.ValueOf(t.UTC()))
		v.Set(reflect.ValueOf(t.Local()))
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		// Walk struct fields (exported only)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			ft := v.Type().Field(i)

			// skip unexported fields
			if ft.PkgPath != "" {
				continue
			}

			normalizeValueToUTC(f)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			normalizeValueToUTC(v.Index(i))
		}
	case reflect.Map:
		// Best-effort: normalize map values when they are settable via a copied value.
		// (We avoid being too clever here; most DTOs are structs anyway.)
		iter := v.MapRange()
		for iter.Next() {
			val := iter.Value()
			// only normalize if the value is a pointer/struct/time and can be updated meaningfully
			normalizeValueToUTC(val)
		}
	default:
		// primitives: nothing to do
	}
}
