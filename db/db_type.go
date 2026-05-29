package db

import (
	"time"

	"github.com/aarondl/opt/null"
)

var (
	NullString = null.Val[string]{}
	NullInt32  = null.Val[int32]{}
	NullInt64  = null.Val[int64]{}
	NullTime   = null.Val[time.Time]{}
)
