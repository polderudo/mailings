// Package snippet stellt vordefinierte HTML-Bausteine bereit, die im
// Mailing-Template-Editor (Quill) per Toolbar-Picker an der aktuellen
// Cursor-Position eingefügt werden können.
//
// Der Inhalt ist bewusst als Code-Konstante gepflegt — sollte sich der Bedarf
// hin zu administrierbaren Schnipseln ändern, kann die Liste später durch
// eine DB-Tabelle (z. B. mail_snippet) ersetzt werden, ohne dass Editor oder
// View etwas davon mitbekommen.
package snippet

// Snippet ist ein benannter HTML-Baustein.
type Snippet struct {
	ID   string
	Name string
	HTML string
}

var snippets = []Snippet{
	{
		ID:   "footer-scp",
		Name: "Footer SCP",
		// Bewusst kein <hr>: Quill v2 hat dafür keinen Standard-Block-Blot,
		// das führt nach dem Einfügen zu DOM-Inkonsistenzen beim nächsten
		// Enter ("insertBefore … not a child of this node"). Stattdessen ein
		// Absatz mit Em-Dash als optische Trennung.
		HTML: `<p>—</p>` +
			`<p><b>Service Center Projekt Vertriebs GmbH</b><br>` +
			`E-Mail: <a href="mailto:info@sid-card.de">info@sid-card.de</a> | Telefon: 089 85601-0<br>` +
			`<a href="https://www.sid-card.de">www.sid-card.de</a> | ` +
			`<a href="https://cardy.cloud">cardy.cloud</a> | ` +
			`<a href="https://cardy.cloud/datenschutz">cardy.cloud/datenschutz</a></p>`,
	},
	{
		ID:   "greeting-de",
		Name: "Begrüßung (Platzhalter)",
		HTML: `<p>Sehr geehrte Damen und Herren,</p>` +
			`<p>(Hier den eigentlichen Mailing-Text einfügen.)</p>`,
	},
}

// All gibt die aktuell vordefinierten Snippets in stabiler Reihenfolge zurück.
func All() []Snippet {
	out := make([]Snippet, len(snippets))
	copy(out, snippets)
	return out
}
