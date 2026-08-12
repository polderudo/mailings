package postmark

import (
	"encoding/json"
	"strings"
	"testing"
)

// Bei einer reinen Text-Mail darf KEIN HtmlBody im JSON stehen — ein leerer
// HtmlBody ließe Postmark eine Multipart-Mail mit leerem HTML-Teil bauen.
func TestBulkRequestOmitsEmptyHTMLBody(t *testing.T) {
	b, err := json.Marshal(bulkRequest{From: "a@b.de", Subject: "s", TextBody: "hallo"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "HtmlBody") {
		t.Errorf("HtmlBody darf bei Text-Mails nicht im Payload stehen: %s", b)
	}
	if !strings.Contains(string(b), `"TextBody":"hallo"`) {
		t.Errorf("TextBody fehlt im Payload: %s", b)
	}
}

// Der Abmelde-Footer muss genau einmal im Text stehen: fehlt der Token, wird er
// angehängt; ist er bereits vorhanden, bleibt der Body unverändert.
func TestWithUnsubscribeFooterText(t *testing.T) {
	got := withUnsubscribeFooterText("Hallo Welt\n\n")
	if strings.Count(got, unsubscribeToken) != 1 {
		t.Errorf("erwartet genau ein Unsubscribe-Token, bekam:\n%s", got)
	}
	if !strings.HasPrefix(got, "Hallo Welt") {
		t.Errorf("Body-Anfang verändert: %q", got)
	}

	own := "Abmelden: {{{ pm:unsubscribe }}}"
	if withUnsubscribeFooterText(own) != own {
		t.Error("Body mit eigenem Token darf nicht ergänzt werden")
	}
}
