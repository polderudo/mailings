package validate

import (
	"app/api/api_error"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func ValidateData[T any](c echo.Context) (*T, error) {
	l := new(T)
	if err := c.Bind(l); err != nil {
		return nil, fmt.Errorf("error binding data: %w", err)
	}

	//normalizeTimesToUTC(l)

	if err := c.Validate(l); err != nil {
		return nil, err
	}

	return l, nil
}

func ValidationError(c echo.Context, err error) error {
	_, ok := err.(validator.ValidationErrors)
	if ok {
		return api_error.NewValidation(c).Msg(err.(validator.ValidationErrors).Error())
	} else {
		return api_error.NewValidation(c).Msg(err.Error())
	}
}

type CustomValidator struct {
	validator *validator.Validate
}

// Validate validates the structs
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
