package cron

import (
	"app/conf"
	"app/db"
	"app/mail/postmark"
	"app/mq"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/go-co-op/gocron/v2"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// InitCron startet den Hintergrund-Scheduler. Aktuell ein Job: alle 30 s den
// Versandfortschritt laufender Mailings aus Postmark nachziehen.
func InitCron() error {
	if conf.C.Postmark.ServerToken == "" {
		slog.Warn("cron: Postmark.ServerToken leer — Bulk-Status-Poller wird nicht gestartet")
		return nil
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("error creating scheduler: %w", err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(30*time.Second),
		gocron.NewTask(pollSendingMailings),
		gocron.WithIntervalFromCompletion(),
	)
	if err != nil {
		return fmt.Errorf("error creating pollSendingMailings cron job: %w", err)
	}

	// Postmark-Suppression-Liste (Bounces/Spam/Unsubscribes) alle 15 min in die
	// globale mail_blacklist spiegeln. Läuft direkt beim Start einmal, damit die
	// Blacklist nach einem Deploy zeitnah aktuell ist.
	_, err = s.NewJob(
		gocron.DurationJob(60*time.Minute),
		gocron.NewTask(syncBlacklist),
		gocron.WithIntervalFromCompletion(),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return fmt.Errorf("error creating syncBlacklist cron job: %w", err)
	}

	s.Start()
	slog.Info("cron started", "jobs", "pollSendingMailings(30s), syncBlacklist(60m)")
	return nil
}

// syncBlacklist ist der Cron-Wrapper um postmark.SyncSuppressions.
func syncBlacklist() {
	if _, err := postmark.SyncSuppressions(context.Background()); err != nil {
		slog.Error("cron: blacklist sync", "err", err)
	}
}

// pollSendingMailings sucht Mailings mit unserem Status "sending" und zieht für
// jedes den Postmark-Bulk-Status nach. Erreicht ein Bulk-Request "Completed"
// (bzw. 100 %), setzen wir unseren Status auf "done".
func pollSendingMailings() {
	ctx := context.Background()

	mailings, err := mq.Mailings.Query(
		sm.Where(mq.Mailings.Columns.Status.EQ(psql.Arg("sending"))),
	).All(ctx, db.DBob)
	if err != nil {
		slog.Error("cron: query sending mailings", "err", err)
		return
	}
	if len(mailings) == 0 {
		return
	}

	client := postmark.NewClient(conf.C.Postmark.ServerToken)
	for _, m := range mailings {
		if m.PostmarkBulkRequestID == "" {
			continue
		}

		// Bei gechunkten Versänden stehen mehrere ids komma-separiert; wir
		// aggregieren über alle Requests (Summe Nachrichten, Min-Fortschritt).
		ids := strings.Split(m.PostmarkBulkRequestID, ",")
		var (
			total      int32
			minPct     = 100.0
			lastStatus string
			lastSubmit string
			allDone    = true
			anyOK      bool
		)
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			st, err := client.GetBulkStatus(ctx, id)
			if err != nil {
				slog.Error("cron: get bulk status", "err", err, "mailing", m.ID, "bulk", id)
				allDone = false
				continue
			}
			anyOK = true
			total += st.TotalMessages
			if st.PercentageCompleted < minPct {
				minPct = st.PercentageCompleted
			}
			lastStatus = st.Status
			lastSubmit = st.SubmittedAt
			if st.Status != "Completed" && st.PercentageCompleted < 100 {
				allDone = false
			}
		}
		if !anyOK {
			continue
		}

		setter := &mq.MailingSetter{
			PostmarkStatus:              omit.From(lastStatus),
			PostmarkTotalMessages:       omit.From(total),
			PostmarkPercentageCompleted: omit.From(minPct),
			UpdatedAt:                   omitnull.From(time.Now()),
		}
		if t, perr := time.Parse(time.RFC3339, lastSubmit); perr == nil {
			setter.PostmarkSubmittedAt = omitnull.From(t)
		}
		if allDone {
			setter.Status = omit.From("done")
			setter.FinishedAt = omitnull.From(time.Now())
		}
		if err := m.Update(ctx, db.DBob, setter); err != nil {
			slog.Error("cron: update mailing status", "err", err, "mailing", m.ID)
		}
	}
}
