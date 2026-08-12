// Package format hält die Format-Kennung einer Mail-Vorlage bzw. eines
// Mailings. Sie entscheidet, wie der gespeicherte Body vor dem Versand
// aufbereitet wird:
//
//   - HTML: der TinyMCE-Inhalt wird in das newsletter_html-Gerüst gehüllt und
//     als HtmlBody an Postmark geschickt.
//   - Text: der eingegebene Text geht 1:1 als TextBody an Postmark, ohne
//     HTML-Anteil.
//
// Die Konstanten entsprechen den Werten der Spalten mail_template.format und
// mailing.format.
package format

const (
	HTML = "html"
	Text = "text"
)

// Normalize bildet einen beliebigen (auch leeren oder unbekannten) Wert auf
// eine der beiden gültigen Kennungen ab. Default ist HTML — Bestandsdaten ohne
// gesetztes Format sind HTML-Newsletter.
func Normalize(v string) string {
	if v == Text {
		return Text
	}
	return HTML
}

// IsText meldet, ob der Body als reine Text-Mail zu versenden ist.
func IsText(v string) bool { return Normalize(v) == Text }
