package api

import (
	"database/sql/driver"
	"github.com/go-playground/validator/v10"
	"reflect"
)

// CustomValidator struct
type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the structs
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func ValidateValuer(field reflect.Value) interface{} {

	if valuer, ok := field.Interface().(driver.Valuer); ok {

		val, err := valuer.Value()
		if err == nil {
			return val
		}
		// handle the error how you want
	}

	return nil
}

func oneFieldSet(fl validator.FieldLevel) bool {
	field := fl.Field()

	currentField, _, ok := fl.GetStructFieldOK()
	if !ok {
		return false
	}

	defaultF := field.Interface() == reflect.Zero(field.Type()).Interface()
	defaultC := currentField.Interface() == reflect.Zero(currentField.Type()).Interface()

	if !defaultF || !defaultC {
		return true
	}

	return false
}
