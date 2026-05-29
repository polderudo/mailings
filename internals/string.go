package internals

import (
	"fmt"
	"strings"
)

func PadLeft(strToPad string, padChar string, lenStr int) string {
	if len(strToPad) >= lenStr {
		return strToPad
	}
	return strings.Repeat(padChar, lenStr-len(strToPad)) + strToPad
}

func Pad0Left(strToPad string, lenStr int) string {
	return fmt.Sprintf("%0*s", lenStr, strToPad)
}

func PadRight(str, fillup string, length int) string {
	formatString := fmt.Sprintf("%%-%ds", length)
	return fmt.Sprintf(formatString, str+strings.Repeat(fillup, length-len(str)))
}

func Substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func IsStringComposedOf(str string, s string, isEmptyTrue bool) bool {
	if str == "" && isEmptyTrue {
		return true
	}
	l := len(s)
	for i := 0; i < len(str); i += l {
		if i+l > len(str) {
			return false
		}
		if str[i:i+l] != s {
			return false
		}
	}
	return true
}

// ContainsSubstring to check if any string in the list contains the given string
func ContainsSubstring(s string, list []string) bool {
	for _, item := range list {
		if strings.Contains(strings.ToLower(item), strings.ToLower(s)) {
			return true
		}
	}
	return false
}

// RemoveCutset entfernt alle Vorkommen der Zeichen eines Schnittmengen-Zeichensatzes (cutset) aus einem String.
func RemoveCutset(s, cutset string) string {
	replacer := strings.NewReplacer(createReplacerPairs(cutset)...)
	return replacer.Replace(s)
}

// createReplacerPairs erstellt ein Array von Ersetzungen für strings.NewReplacer.
// Jedes Zeichen im cutset wird als zu entfernendes Zeichen definiert.
func createReplacerPairs(cutset string) []string {
	pairs := make([]string, 0, len(cutset)*2)
	for _, char := range cutset {
		pairs = append(pairs, string(char), "")
	}
	return pairs
}
