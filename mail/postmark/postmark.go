package postmark

import (
	"app/conf"
	"app/db"
	"app/mq"
	"app/templates"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"golang.org/x/net/idna"
)

const (
	defaultBaseURL = "https://api.postmarkapp.com"
	// maxBulkMessages ist das Postmark-Limit pro /email/bulk-Request.
	maxBulkMessages = 5000
	httpTimeout     = 60 * time.Second
)

// unsubscribeToken ist der Postmark-Merge-Token, den Broadcast-Message-Streams
// pro Empfänger durch eine eindeutige One-Click-Unsubscribe-URL ersetzen. Er
// MUSS unverändert (dreifache geschweifte Klammern inkl. Leerzeichen) im
// HtmlBody landen. Siehe:
// https://postmarkapp.com/support/article/1208-how-to-add-an-unsubscribe-link
const unsubscribeToken = "{{{ pm:unsubscribe }}}"

// unsubscribeFooterHTML wird an jeden ausgehenden Broadcast-HTML-Body angehängt,
// damit jede Mail garantiert einen eindeutigen Abmelde-Link enthält. Text /
// Styling hier zentral editierbar. Das href nutzt den Postmark-Unsubscribe-
// Merge-Token, den Postmark auf Broadcast-Streams pro Empfänger auflöst.
const unsubscribeFooterHTML = `<p style="font-size:12px;color:#888;margin:24px"><strong>legal disclaimer</strong><br><br>Es liegt nicht in unserer Absicht, Ihnen unerwünschte Informationen per E-Mail zukommen zu lassen. Sie erhalten diese E-Mail als unser Kunde oder weil wir Sie als Interessenten für unsere Newsletter zum Thema Schule/Bildung in unserer Datenbank führen. Falls Sie zukünftig keine Information mehr erhalten möchten, können Sie sich vom Newsletter einfach <a href="{{{ pm:unsubscribe }}}">hier abmelden</a> und wir löschen Ihre Daten umgehend und vollständig.</p>`

// withUnsubscribeFooter hängt den Abmelde-Footer an den HTML-Body an. Idempotent:
// enthält der Body den Unsubscribe-Token bereits (z. B. manuell im Template
// platziert oder bereits angehängt), bleibt er unverändert. Liegt ein
// vollständiges HTML-Dokument vor, wird der Footer direkt vor das schließende
// </body>-Tag eingefügt, sonst angehängt.
func withUnsubscribeFooter(htmlBody string) string {
	if strings.Contains(htmlBody, unsubscribeToken) {
		return htmlBody
	}
	lower := strings.ToLower(htmlBody)
	if idx := strings.LastIndex(lower, "</body>"); idx >= 0 {
		return htmlBody[:idx] + unsubscribeFooterHTML + htmlBody[idx:]
	}
	return htmlBody + unsubscribeFooterHTML
}

// Client kapselt die Postmark Bulk-Email-API.
// Wird per `conf.C.Postmark.ServerToken` initialisiert.
type Client struct {
	ServerToken string
	BaseURL     string
	HTTP        *http.Client
}

func NewClient(serverToken string) *Client {
	base := conf.C.Postmark.ApiBaseURL
	if base == "" {
		base = defaultBaseURL
	}
	base = strings.TrimRight(base, "/")
	return &Client{
		ServerToken: serverToken,
		BaseURL:     base,
		HTTP:        &http.Client{Timeout: httpTimeout},
	}
}

// --- Request/Response-Typen -------------------------------------------------

type bulkMessage struct {
	To            string         `json:"To"`
	TemplateModel map[string]any `json:"TemplateModel,omitempty"`
}

type bulkRequest struct {
	From          string        `json:"From"`
	ReplyTo       string        `json:"ReplyTo,omitempty"`
	Subject       string        `json:"Subject"`
	HtmlBody      string        `json:"HtmlBody"`
	TextBody      string        `json:"TextBody,omitempty"`
	MessageStream string        `json:"MessageStream,omitempty"`
	TrackOpens    bool          `json:"TrackOpens"`
	TrackLinks    string        `json:"TrackLinks,omitempty"`
	Messages      []bulkMessage `json:"Messages"`
}

// bulkFieldError ist ein einzelner Validierungsfehler aus der Errors-Property
// einer 422-Antwort. Postmark gruppiert sie nach Feldname (z. B. "To"/"From");
// die konkrete Adresse steckt jeweils in Message ("Invalid 'To' address: '…'.").
type bulkFieldError struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
}

// bulkResponse ist die Antwort auf POST /email/bulk (Feld "ID").
type bulkResponse struct {
	ID          string `json:"ID"`
	Status      string `json:"Status"`
	SubmittedAt string `json:"SubmittedAt"`
	// Fehlerform (Non-2xx): bei ErrorCode 11 ("Multiple errors occurred") stehen
	// die feldbezogenen Details in Errors (Feldname → Liste von Fehlern).
	ErrorCode int                         `json:"ErrorCode"`
	Message   string                      `json:"Message"`
	Errors    map[string][]bulkFieldError `json:"Errors"`
}

// formatBulkErrors fasst die feldbezogenen Validierungsfehler einer 422-Antwort
// zu einem loggbaren String zusammen, damit im Log die konkreten Adressen statt
// nur "Multiple errors occurred. Inspect the Errors property" erscheinen.
func formatBulkErrors(errs map[string][]bulkFieldError) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for field, list := range errs {
		for i, e := range list {
			if i >= 5 {
				fmt.Fprintf(&b, " | %s: (+%d weitere)", field, len(list)-5)
				break
			}
			fmt.Fprintf(&b, " | %s: %s", field, e.Message)
		}
	}
	return b.String()
}

// BulkStatus ist die Antwort auf GET /email/bulk/{id} (Feld "Id").
type BulkStatus struct {
	ID                  string  `json:"Id"`
	Status              string  `json:"Status"`
	SubmittedAt         string  `json:"SubmittedAt"`
	TotalMessages       int32   `json:"TotalMessages"`
	PercentageCompleted float64 `json:"PercentageCompleted"`
	Subject             string  `json:"Subject"`
}

// emailToASCII wandelt den Domain-Teil einer E-Mail-Adresse in seine ASCII-/
// Punycode-Form (IDNA) um, falls er Unicode-Zeichen (z. B. Umlaute) enthält.
// Der Local-Part bleibt unverändert. Postmark akzeptiert im Domain-Teil nur
// ASCII; eine Unicode-Domain wie "schön-schule.de" muss als
// "xn--schn-schule-9hb.de" verschickt werden, sonst lehnt Postmark die Adresse
// mit 422/ErrorCode 11 ab und der gesamte Bulk-Chunk schlägt fehl.
//
// Existiert kein "@", wird die Adresse unverändert zurückgegeben (die
// Adress-Validierung an anderer Stelle fängt das ab). Lässt sich der Domain-Teil
// nicht in eine gültige IDNA-/ASCII-Form konvertieren, wird ein Fehler
// zurückgegeben, damit der Aufrufer die Adresse als ungültig behandeln kann.
func emailToASCII(addr string) (string, error) {
	at := strings.LastIndex(addr, "@")
	if at < 0 {
		return addr, nil
	}
	local, domain := addr[:at], addr[at+1:]
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return "", err
	}
	return local + "@" + ascii, nil
}

// --- Versand ----------------------------------------------------------------

// SendBulk bereitet das Mailing auf (HTML-Wrapper) und schickt es über die
// Postmark Bulk-API. Die zurückgegebene bulk-request-id wird auf dem Mailing
// gespeichert; der Versandfortschritt wird anschließend vom Cron-Poller über
// GetBulkStatus nachgezogen.
func (c *Client) SendBulk(ctx context.Context, mailingID int32) error {
	m, err := mq.FindMailing(ctx, db.DBob, mailingID)
	if err != nil || m == nil {
		return errors.New("mailing not found")
	}
	if m.Status != "draft" {
		return fmt.Errorf("mailing %d not in draft status (is %q)", mailingID, m.Status)
	}

	d, err := mq.FindMailDomain(ctx, db.DBob, m.DomainID)
	if err != nil || d == nil {
		return c.fail(ctx, m, "domain not found")
	}

	recipients, err := mq.MailListRecipients.Query(
		sm.Where(mq.MailListRecipients.Columns.ListID.EQ(psql.Arg(m.ListID))),
		sm.OrderBy(mq.MailListRecipients.Columns.Email),
	).All(ctx, db.DBob)
	if err != nil {
		return c.fail(ctx, m, "load recipients: "+err.Error())
	}
	if len(recipients) == 0 {
		return c.fail(ctx, m, "list has no recipients")
	}

	htmlBody, err := templates.RenderNewsletterHTML(m.SubjectSnapshot, m.BodySnapshot, m.SubjectSnapshot)
	if err != nil {
		return c.fail(ctx, m, "render html: "+err.Error())
	}
	// Abmelde-Footer mit Postmark-Unsubscribe-Token zentral anhängen, damit jede
	// versendete Mail einen eindeutigen Abmelde-Link enthält (Broadcast-Stream).
	htmlBody = withUnsubscribeFooter(htmlBody)

	// Domain-Teil der Absender-Adresse in Punycode wandeln, falls Unicode (Umlaut)
	// enthalten ist — Postmark akzeptiert im Domain-Teil nur ASCII. Lässt sich die
	// Absender-Domain nicht konvertieren, kann gar nichts versendet werden → fail.
	fromEmail, err := emailToASCII(d.FromEmail)
	if err != nil {
		return c.fail(ctx, m, "ungültige Absender-Domain: "+err.Error())
	}
	from := fromEmail
	if strings.TrimSpace(d.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", d.FromName, fromEmail)
	}

	// Empfänger vor dem Versand validieren. Eine einzige ungültige Adresse lässt
	// Postmark den GESAMTEN Bulk-Request mit 422/ErrorCode 11 ablehnen — alle
	// Empfänger im Chunk gehen dann verloren (0 statt n-1 versendet). Deshalb
	// filtern wir kaputte Adressen vorab heraus, markieren sie als failed und
	// versenden nur die gültigen.
	messages := make([]bulkMessage, 0, len(recipients))
	var invalid []*mq.MailListRecipient
	for _, r := range recipients {
		addr := strings.TrimSpace(r.Email)
		if _, perr := netmail.ParseAddress(addr); perr != nil {
			invalid = append(invalid, r)
			continue
		}
		// Domain-Teil in Punycode wandeln, falls Unicode (Umlaut) enthalten ist —
		// netmail.ParseAddress akzeptiert Unicode-Domains, Postmark jedoch nicht.
		// Eine nicht in gültiges IDNA konvertierbare Domain ist für Postmark
		// unbrauchbar und wird daher wie eine ungültige Adresse aussortiert.
		ascii, aerr := emailToASCII(addr)
		if aerr != nil {
			invalid = append(invalid, r)
			continue
		}
		addr = ascii
		msg := bulkMessage{To: addr}
		if r.Forename != "" || r.Lastname != "" {
			msg.TemplateModel = map[string]any{
				"forename": r.Forename,
				"lastname": r.Lastname,
			}
		}
		messages = append(messages, msg)
	}

	if len(invalid) > 0 {
		markInvalidRecipients(ctx, mailingID, invalid)
		slog.Warn("postmark bulk: ungültige Empfänger-Adressen übersprungen",
			"mailing", mailingID, "skipped", len(invalid), "valid", len(messages),
			"examples", sampleEmails(invalid, 10))
	}
	if len(messages) == 0 {
		return c.fail(ctx, m, "keine gültigen Empfänger-Adressen in der Liste")
	}

	// Postmark erlaubt max. 500 Nachrichten pro /email/bulk-Request. Bei größeren
	// Listen wird in 500er-Chunks sequentiell versendet; die zurückgegebenen
	// bulk-request-ids werden komma-separiert gespeichert (für die Status-Abfrage).
	if len(messages) > maxBulkMessages {
		slog.Warn("postmark bulk: recipient list exceeds single-request limit, chunking",
			"mailing", mailingID, "recipients", len(messages), "limit", maxBulkMessages)
	}

	var requestIDs []string
	var firstStatus, firstSubmittedAt string
	for start := 0; start < len(messages); start += maxBulkMessages {
		end := start + maxBulkMessages
		if end > len(messages) {
			end = len(messages)
		}
		req := bulkRequest{
			From:          from,
			Subject:       m.SubjectSnapshot,
			HtmlBody:      htmlBody,
			MessageStream: d.PostmarkStreamID,
			TrackOpens:    conf.C.Postmark.TrackOpens,
			TrackLinks:    conf.C.Postmark.TrackLinks,
			Messages:      messages[start:end],
		}
		resp, err := c.postBulk(ctx, req)
		if err != nil {
			return c.fail(ctx, m, err.Error())
		}
		requestIDs = append(requestIDs, resp.ID)
		if firstStatus == "" {
			firstStatus = resp.Status
			firstSubmittedAt = resp.SubmittedAt
		}
	}

	now := time.Now()
	setter := &mq.MailingSetter{
		Status:                omit.From("sending"),
		PostmarkBulkRequestID: omit.From(strings.Join(requestIDs, ",")),
		PostmarkStatus:        omit.From(firstStatus),
		StartedAt:             omitnull.From(now),
		UpdatedAt:             omitnull.From(now),
	}
	if t, perr := time.Parse(time.RFC3339, firstSubmittedAt); perr == nil {
		setter.PostmarkSubmittedAt = omitnull.From(t)
	}
	if err := m.Update(ctx, db.DBob, setter); err != nil {
		return err
	}
	slog.Info("postmark bulk submitted", "mailing", mailingID,
		"requests", len(requestIDs), "recipients", len(messages))
	return nil
}

// postBulk führt einen einzelnen POST /email/bulk aus.
func (c *Client) postBulk(ctx context.Context, req bulkRequest) (*bulkResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal bulk request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/email/bulk", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("postmark bulk request: %w", err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	var resp bulkResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("postmark bulk: status %d, unparseable response: %s", res.StatusCode, string(raw))
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("postmark bulk: status %d, error %d: %s%s",
			res.StatusCode, resp.ErrorCode, resp.Message, formatBulkErrors(resp.Errors))
	}
	if resp.ID == "" {
		return nil, fmt.Errorf("postmark bulk: empty request id in response: %s", string(raw))
	}
	return &resp, nil
}

// GetBulkStatus liest den Verarbeitungsstatus eines Bulk-Requests.
func (c *Client) GetBulkStatus(ctx context.Context, bulkRequestID string) (*BulkStatus, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/email/bulk/"+bulkRequestID, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq)

	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("postmark bulk status request: %w", err)
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("postmark bulk status: http %d: %s", res.StatusCode, string(raw))
	}
	var st BulkStatus
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, fmt.Errorf("postmark bulk status: unparseable response: %s", string(raw))
	}
	return &st, nil
}

func (c *Client) setHeaders(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	r.Header.Set("X-Postmark-Server-Token", c.ServerToken)
}

// markInvalidRecipients schreibt den failed-Status auf die vor dem Versand
// herausgefilterten Empfänger, damit sie im UI (Recipients-Tabelle) als
// fehlgeschlagen sichtbar sind und beim nächsten Mailing korrigiert werden können.
func markInvalidRecipients(ctx context.Context, mailingID int32, recipients []*mq.MailListRecipient) {
	for _, r := range recipients {
		if err := r.Update(ctx, db.DBob, &mq.MailListRecipientSetter{
			LastStatus:    omit.From("failed"),
			LastError:     omit.From("ungültige E-Mail-Adresse"),
			LastMailingID: omitnull.From(mailingID),
		}); err != nil {
			slog.Error("postmark bulk: mark invalid recipient", "err", err,
				"mailing", mailingID, "recipient", r.ID)
		}
	}
}

// sampleEmails liefert bis zu n Adressen für eine knappe Log-Ausgabe.
func sampleEmails(recipients []*mq.MailListRecipient, n int) []string {
	out := make([]string, 0, n)
	for _, r := range recipients {
		if len(out) >= n {
			break
		}
		out = append(out, r.Email)
	}
	return out
}

// fail markiert das Mailing als fehlgeschlagen und gibt einen Fehler zurück. Der
// reason-Text wird zusätzlich in postmark_status_error abgelegt, damit der
// Fehlerfall (z. B. die feldbezogenen Postmark-Validierungsfehler) im Frontend
// nachvollziehbar ist und nicht nur im Log steht.
func (c *Client) fail(ctx context.Context, m *mq.Mailing, reason string) error {
	now := time.Now()
	_ = m.Update(ctx, db.DBob, &mq.MailingSetter{
		Status:              omit.From("failed"),
		PostmarkStatusError: omit.From(reason),
		FinishedAt:          omitnull.From(now),
		UpdatedAt:           omitnull.From(now),
	})
	return errors.New(reason)
}
