package postmark

import (
	"app/db"
	"app/mq"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// Client kapselt die Postmark Broadcasts API.
// Wird per `conf.C.Postmark.ServerToken` initialisiert.
type Client struct {
	ServerToken string
}

func NewClient(serverToken string) *Client {
	return &Client{ServerToken: serverToken}
}

// SendBroadcast versendet das Mailing an alle Empfänger der verknüpften Liste.
// Der letzte Status wird direkt auf mail_list_recipient (last_status,
// last_postmark_message_id, last_error, last_sent_at, last_mailing_id)
// fortgeschrieben. Die Aggregate (sent_count/failed_count) landen auf mailing.
//
// Die eigentliche Postmark-HTTP-Anbindung ist noch ein Stub und mit TODO markiert.
func (c *Client) SendBroadcast(ctx context.Context, mailingID int32) error {
	m, err := mq.FindMailing(ctx, db.DBob, mailingID)
	if err != nil || m == nil {
		return errors.New("mailing not found")
	}

	now := time.Now()
	if err := m.Update(ctx, db.DBob, &mq.MailingSetter{
		Status:    omit.From("sending"),
		StartedAt: omitnull.From(now),
		UpdatedAt: omitnull.From(now),
	}); err != nil {
		return err
	}

	recipients, err := mq.MailListRecipients.Query(
		sm.Where(mq.MailListRecipients.Columns.ListID.EQ(psql.Arg(m.ListID))),
		sm.OrderBy(mq.MailListRecipients.Columns.Email),
	).All(ctx, db.DBob)
	if err != nil {
		return err
	}

	sent := 0
	failed := 0
	for _, r := range recipients {
		// TODO: HTTP-POST zu https://api.postmarkapp.com/email
		//       Header: X-Postmark-Server-Token: c.ServerToken
		//       Body:   { From, To, Subject, HtmlBody, MessageStream }
		sentAt := time.Now()
		if err := r.Update(ctx, db.DBob, &mq.MailListRecipientSetter{
			LastStatus:            omit.From("sent"),
			LastPostmarkMessageID: omit.From(""),
			LastError:             omit.From(""),
			LastSentAt:            omitnull.From(sentAt),
			LastMailingID:         omitnull.From(m.ID),
		}); err != nil {
			slog.Error("update recipient", "err", err, "mailing", mailingID, "email", r.Email)
			failed++
			continue
		}
		sent++
	}

	doneAt := time.Now()
	if err := m.Update(ctx, db.DBob, &mq.MailingSetter{
		Status:      omit.From("done"),
		SentCount:   omit.From(int32(sent)),
		FailedCount: omit.From(int32(failed)),
		FinishedAt:  omitnull.From(doneAt),
		UpdatedAt:   omitnull.From(doneAt),
	}); err != nil {
		return err
	}
	slog.Info("postmark broadcast finished", "mailing", mailingID, "sent", sent, "failed", failed)
	return nil
}
