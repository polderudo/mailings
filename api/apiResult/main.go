package apiResult

import "app/i18n"

type Result map[string]string

func New(message string) Result {
	return Result{"message": message}
}

func NewI18N(key int) Result {
	return Result{"message": i18n.Tr(key)}
}
