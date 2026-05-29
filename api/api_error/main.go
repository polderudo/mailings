package api_error

import (
	"app/constants"
	"app/core_error"
	"app/i18n"
	"app/ilog"
	"app/srvCtx"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Error struct {
	Message        string              `json:"message"`
	Code           constants.ErrorCode `json:"code"`
	srvCtx         *srvCtx.BobContext  `json:"-"`
	httpStatusCode int                 `json:"-"`
	err            error               `json:"-"`
}

func New(c echo.Context) *Error {
	return &Error{
		srvCtx: srvCtx.NewEchoBobCtx(c),
	}
}

func NewWithError(c echo.Context, err error) *Error {
	return &Error{
		srvCtx:         srvCtx.NewEchoBobCtx(c),
		httpStatusCode: http.StatusInternalServerError, //default
		err:            err,
	}
}

func NewValidation(c echo.Context) *Error {
	return &Error{
		srvCtx:         srvCtx.NewEchoBobCtx(c),
		Code:           constants.ErrValidation,
		httpStatusCode: http.StatusBadRequest, //default
	}
}

func NewWithCode(c echo.Context, Code constants.ErrorCode) *Error {
	r := &Error{
		srvCtx:         srvCtx.NewEchoBobCtx(c),
		Code:           Code,
		httpStatusCode: http.StatusBadRequest,
	}
	return r
}

func (e *Error) SetHttpStatusCode(code int) *Error {
	e.httpStatusCode = code
	return e
}

func (e *Error) MsgTr(i18nKey int, args ...any) error {
	m := i18n.Tr(i18nKey, args...)
	return e.Msg(m)
}

func (e *Error) Msg(message string) error {
	e.Message = message

	if e.err != nil {
		if _, ok := e.err.(*core_error.Warning); ok {
			e.httpStatusCode = http.StatusBadRequest
		}
		e.Message = fmt.Sprintf("%s : %s", e.Message, e.err.Error())
		e.srvCtx.Log.Error(message, ilog.Err(e.err))
	} else {
		e.srvCtx.Log.Warn(message, ilog.Err(e.err))
	}

	return e.srvCtx.EchoContext.JSON(e.httpStatusCode, e)
}

func (e *Error) Msgf(message string, params ...interface{}) error {
	return e.Msg(fmt.Sprintf(message, params...))
}
