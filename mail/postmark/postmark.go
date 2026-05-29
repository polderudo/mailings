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
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

const (
	defaultBaseURL = "https://api.postmarkapp.com"
	// maxBulkMessages ist das Postmark-Limit pro /email/bulk-Request.
	maxBulkMessages = 500
	httpTimeout     = 30 * time.Second
)

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

// bulkResponse ist die Antwort auf POST /email/bulk (Feld "ID").
type bulkResponse struct {
	ID          string `json:"ID"`
	Status      string `json:"Status"`
	SubmittedAt string `json:"SubmittedAt"`
	// Fehlerform (Non-2xx):
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
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

	from := d.FromEmail
	if strings.TrimSpace(d.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", d.FromName, d.FromEmail)
	}

	messages := make([]bulkMessage, 0, len(recipients))
	for _, r := range recipients {
		msg := bulkMessage{To: r.Email}
		if r.Forename != "" || r.Lastname != "" {
			msg.TemplateModel = map[string]any{
				"forename": r.Forename,
				"lastname": r.Lastname,
			}
		}
		messages = append(messages, msg)
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
		return nil, fmt.Errorf("postmark bulk: status %d, error %d: %s", res.StatusCode, resp.ErrorCode, resp.Message)
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

// fail markiert das Mailing als fehlgeschlagen und gibt einen Fehler zurück.
func (c *Client) fail(ctx context.Context, m *mq.Mailing, reason string) error {
	now := time.Now()
	_ = m.Update(ctx, db.DBob, &mq.MailingSetter{
		Status:     omit.From("failed"),
		FinishedAt: omitnull.From(now),
		UpdatedAt:  omitnull.From(now),
	})
	return errors.New(reason)
}
