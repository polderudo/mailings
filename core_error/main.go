package core_error

import (
	"errors"
	"fmt"
)

type Warning struct {
	error
}

func (r *Warning) Error() string {
	return r.error.Error()
}

func NewWarning(err error) *Warning {
	return &Warning{
		err,
	}
}

func NewWarningStr(str string, params ...interface{}) *Warning {
	err := errors.New(fmt.Sprintf(str, params...))
	return &Warning{
		err,
	}
}
