// Package address bereinigt und prüft Empfänger-Adressen.
//
// Hintergrund: Aus Excel, Word oder Webseiten kopierte Adressen enthalten
// häufig unsichtbare Zeichen (Zero-Width-Space, Soft-Hyphen, LRM, geschützte
// Leerzeichen). net/mail.ParseAddress akzeptiert sie im lokalen Teil (RFC 6532),
// Postmark lehnt sie dagegen ab und verwirft damit den GESAMTEN Bulk-Request.
package address

import (
	"strings"
	"unicode"
)

// Clean entfernt aus s alle Leerraum-, Steuer- und unsichtbaren Formatzeichen
// (Unicode-Kategorie Cf: U+200B..U+200F, U+2060, U+FEFF, U+00AD u. a.).
// Eine E-Mail-Adresse enthält nie Leerraum, daher wird auch innerer Leerraum
// entfernt, nicht nur der am Rand.
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
}

// atext sind die ohne Quoting erlaubten Zeichen im lokalen Teil (RFC 5322)
// zusätzlich zu Buchstaben und Ziffern. Der Punkt wird separat geprüft.
const atext = "!#$%&'*+/=?^_`{|}~-"

// ValidLocalPart prüft den lokalen Teil (vor dem @) strikt auf ASCII: so
// verlangt es Postmark. Leer, führender/abschließender Punkt, doppelte Punkte
// und alle Zeichen außerhalb von atext sind ungültig.
func ValidLocalPart(local string) bool {
	if local == "" || len(local) > 64 {
		return false
	}
	if local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return false
	}
	for i := 0; i < len(local); i++ {
		c := local[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.':
		case strings.IndexByte(atext, c) >= 0:
		default:
			return false
		}
	}
	return true
}

// ValidLocalPartOf prüft den lokalen Teil einer vollständigen Adresse.
func ValidLocalPartOf(addr string) bool {
	at := strings.LastIndex(addr, "@")
	if at < 0 {
		return false
	}
	return ValidLocalPart(addr[:at])
}
