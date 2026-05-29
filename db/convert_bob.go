package db

import (
	"github.com/aarondl/opt/null"
	"github.com/aarondl/opt/omitnull"
)

func NullVal[T comparable](v T) null.Val[T] {
	var vNull T
	if v == vNull {
		return null.Val[T]{}
	}
	return null.From(v)
}

func NullValP[T comparable](v *T) null.Val[T] {
	if v == nil {
		return null.Val[T]{}
	}
	return NullVal(*v)
}

func OmitNullVal[T comparable](v T) omitnull.Val[T] {
	var vNull T
	if v == vNull {
		return omitnull.Val[T]{}
	}
	return omitnull.From(v)
}

func OmitNullValP[T comparable](v *T) omitnull.Val[T] {
	if v == nil {
		return omitnull.Val[T]{}
	}

	return OmitNullVal(*v)
}
