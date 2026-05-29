package db

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func IsNoRows(err error) bool {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
			return true
		}
	}
	return false
}

func IsSliceEmpty[T any](sl []T) bool {
	if sl == nil {
		return true
	}
	if len(sl) == 0 {
		return true
	}
	return false
}

func IsErrorAndNotNoRows(err error) bool {
	if err != nil {
		if IsNoRows(err) {
			return false
		}
		return true
	}
	return false
}

func ColList(arg ...string) string {
	return strings.Join(arg, ", ")
}
