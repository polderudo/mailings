package i18n

import (
	"app/conf"
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

// DefaultLang is the fallback language when none is set in the context.
var DefaultLang = DE

type contextKey string

const (
	DE = "de"
	EN = "en"
)

var (
	dynKeys []int
)

type LangData struct {
	EN string
	DE string
}

type Data struct {
	LangData
	LangKey string
}

type DynamicLangData map[string]LangData

// LangContextKey is used to store and retrieve the active language in a context.
var LangContextKey interface{} = contextKey("lang")

// WithLang returns a context carrying the given language code.
func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, LangContextKey, lang)
}

// LangFromContext extracts the language code from ctx.
// Falls back to DefaultLang when not set.
func LangFromContext(ctx context.Context) string {
	if ctx != nil {
		if v, ok := ctx.Value(LangContextKey).(string); ok && v != "" {
			return v
		}
	}
	return DefaultLang
}

// Trl translates key using the language stored in ctx.
func Trl(ctx context.Context, key int, args ...interface{}) string {
	return TrLang(LangFromContext(ctx), key, args...)
}

// TrLang translates key using an explicit lang string.
// Use this in plain Go handlers where ctx is not available or lang is already resolved.
func TrLang(lang string, key int, args ...interface{}) string {
	lang = NormalizeLang(lang)
	var str string
	if v, ok := LangMap[key]; ok {
		switch lang {
		case DE:
			if v.DE != "" {
				str = v.DE
			} else {
				str = v.EN
			}
		default:
			str = v.EN
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(str, args...)
	}
	return str
}

// Tr translates key using the system default language.
func Tr(key int, args ...interface{}) string {
	return TrLang(conf.C.SystemLangISOCode, key, args...)
}

func NormalizeLang(lang string) string {
	trimmed := strings.ToLower(strings.TrimSpace(lang))
	if trimmed == "" {
		trimmed = strings.ToLower(strings.TrimSpace(conf.C.SystemLangISOCode))
	}
	switch trimmed {
	case DE:
		return DE
	case EN:
		return EN
	default:
		return EN
	}
}

func IsSupportedLang(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case DE, EN:
		return true
	default:
		return false
	}
}

// TimeFormatString returns the default time formatting string for a given language.
func TimeFormatString(lang string) string {
	switch lang {
	case DE:
		return "02.01.2006 15:04:05"
	default:
		return "2006-01-02 15:04:05"
	}
}

func AddDynKey(langKey int, data LangData) {
	if !slices.Contains(dynKeys, langKey) {
		dynKeys = append(dynKeys, langKey)
	}
	v := LangMap[langKey]
	v.LangData = data
	LangMap[langKey] = v
}

func GetDynKeys() DynamicLangData {
	r := make(DynamicLangData)
	for _, k := range dynKeys {
		v := LangMap[k]
		r[v.LangKey] = v.LangData
	}
	return r
}
