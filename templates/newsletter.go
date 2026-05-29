package templates

import (
	"app/conf"
	"bytes"
	_ "embed"
	"html/template"
	"time"
)

//go:embed mail/newsletter_html.html
var newsletterHTMLSource string

// NewsletterData ist das Datenmodell für das Outlook-kompatible HTML-Gerüst,
// in das der reine TinyMCE-Content vor dem Versand eingebettet wird.
type NewsletterData struct {
	Subject     string
	PreviewText string
	Content     template.HTML // vertrauenswürdiges Operator-HTML (TinyMCE-Inhalt + inline base64-Bilder)
	CompanyName string
	Year        int
}

// newsletterTpl wird beim Paket-Init geparst (unabhängig von InitTemplates) und
// nutzt bewusst html/template, getrennt von der text/template-Registry in main.go.
var newsletterTpl = template.Must(template.New("newsletter_html").Parse(newsletterHTMLSource))

// RenderNewsletterHTML hüllt den vertrauenswürdigen inneren Content in ein
// vollständiges, sehr E-Mail-kompatibles HTML-Dokument (tabellenbasiert, MSO-
// Conditional-Comments). Der Content wird als template.HTML verbatim eingesetzt,
// da er bereits aufbereitetes Operator-HTML ist.
func RenderNewsletterHTML(subject, contentHTML, previewText string) (string, error) {
	var buf bytes.Buffer
	err := newsletterTpl.Execute(&buf, NewsletterData{
		Subject:     subject,
		PreviewText: previewText,
		Content:     template.HTML(contentHTML),
		CompanyName: conf.C.Company.Name,
		Year:        time.Now().Year(),
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
