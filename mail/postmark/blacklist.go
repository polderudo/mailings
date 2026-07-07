package postmark

import (
	"app/conf"
	"app/db"
	"app/mq"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	netmail "net/mail"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// Sentinel-Fehler für das manuelle Sperren einer Adresse (AddToBlacklist), damit
// der Aufrufer sie sauber in lokalisierte Toast-Meldungen übersetzen kann.
var (
	// ErrBlacklistInvalidEmail: die übergebene Adresse ist keine gültige
	// E-Mail-Adresse.
	ErrBlacklistInvalidEmail = errors.New("blacklist: ungültige E-Mail-Adresse")
	// ErrBlacklistAlreadyExists: die Adresse steht bereits (egal ob postmark oder
	// manuell) in der globalen Blacklist.
	ErrBlacklistAlreadyExists = errors.New("blacklist: Adresse bereits gesperrt")
)

// ManualReason ist der Grund-Wert, unter dem manuell gesperrte Adressen in der
// Blacklist geführt werden.
const ManualReason = "Manual"

// AddToBlacklist nimmt eine Adresse manuell in die globale Blacklist auf
// (source='manual', reason='Manual'). Manuelle Einträge tragen keinen Postmark-
// Stream und bleiben beim Suppression-Sync unangetastet; SendBulk filtert sie
// über LoadBlacklistSet trotzdem vor jedem Versand heraus. Ein Postmark-Aufruf
// ist nicht nötig — die Sperre wirkt rein lokal.
//
// Ist die Adresse bereits gesperrt (unique-Index lower(email)), wird
// ErrBlacklistAlreadyExists zurückgegeben; ist sie keine gültige Adresse,
// ErrBlacklistInvalidEmail.
func AddToBlacklist(ctx context.Context, email string) (*mq.MailBlacklist, error) {
	addr := strings.TrimSpace(email)
	if _, perr := netmail.ParseAddress(addr); perr != nil {
		return nil, ErrBlacklistInvalidEmail
	}

	// ON CONFLICT (lower(email)) DO NOTHING: kollidiert die Adresse mit einem
	// bereits vorhandenen Eintrag (postmark oder manuell), liefert RETURNING keine
	// Zeile → sql.ErrNoRows. Das behandeln wir als ErrBlacklistAlreadyExists.
	row, err := mq.MailBlacklists.Insert(&mq.MailBlacklistSetter{
		Email:  omit.From(addr),
		Reason: omit.From(ManualReason),
		Source: omit.From("manual"),
	}, im.OnConflict(psql.Raw("lower(email)")).DoNothing()).One(ctx, db.DBob)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBlacklistAlreadyExists
		}
		return nil, fmt.Errorf("insert mail_blacklist: %w", err)
	}
	slog.Info("blacklist: address added manually", "email", row.Email)
	return row, nil
}

// SyncResult fasst das Ergebnis eines Suppression-Syncs zusammen.
type SyncResult struct {
	Added   int // neu in mail_blacklist aufgenommen
	Removed int // entfernt, weil in Postmark reaktiviert
	Total   int // aktuell von Postmark gesperrte Adressen (alle aktiven Streams)
}

// dumpEntry hält eine Suppression zusammen mit dem Stream, aus dem sie stammt.
type dumpEntry struct {
	sup    Suppression
	stream string
}

// SyncSuppressions spiegelt die Postmark-Suppression-Listen aller aktiven
// Absender-Domains in die globale mail_blacklist. Vorgehen:
//
//  1. distinct aktive postmark_stream_id aus mail_domain ermitteln,
//  2. je Stream den Suppressions-Dump abrufen und über lower(email) vereinen,
//  3. fehlende Adressen als source='postmark' einfügen,
//  4. source='postmark'-Zeilen löschen, die nicht mehr im Dump stehen
//     (in Postmark reaktiviert).
//
// source='manual' gesetzte Einträge bleiben in allen Schritten unangetastet.
func SyncSuppressions(ctx context.Context) (*SyncResult, error) {
	if conf.C.Postmark.ServerToken == "" {
		return nil, errors.New("postmark: ServerToken nicht konfiguriert")
	}

	streams, err := activeStreamIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(streams) == 0 {
		return nil, errors.New("postmark: keine aktiven Absender-Domains mit Stream-ID")
	}

	client := NewClient(conf.C.Postmark.ServerToken)

	// (2) Dump je Stream einsammeln und über lower(email) vereinen. Taucht eine
	// Adresse in mehreren Streams auf, gewinnt der erste Treffer (für die globale
	// Blacklist reicht ein Grund/Stream zur Anzeige).
	current := make(map[string]dumpEntry)
	for _, s := range streams {
		sups, derr := client.DumpSuppressions(ctx, s)
		if derr != nil {
			return nil, fmt.Errorf("postmark suppressions dump (stream %q): %w", s, derr)
		}
		for _, su := range sups {
			key := normalizeEmail(su.EmailAddress)
			if key == "" {
				continue
			}
			if _, ok := current[key]; !ok {
				current[key] = dumpEntry{sup: su, stream: s}
			}
		}
	}

	// Bestehende Blacklist laden (alle Quellen — die unique-Index liegt global
	// auf lower(email), daher dürfen wir keine bereits existierende Adresse
	// erneut einfügen).
	existing, err := mq.MailBlacklists.Query().All(ctx, db.DBob)
	if err != nil {
		return nil, fmt.Errorf("load mail_blacklist: %w", err)
	}
	byEmail := make(map[string]*mq.MailBlacklist, len(existing))
	for _, e := range existing {
		byEmail[normalizeEmail(e.Email)] = e
	}

	// (3) fehlende Adressen einfügen.
	added := 0
	for key, ent := range current {
		if _, ok := byEmail[key]; ok {
			continue // bereits vorhanden (postmark oder manual) → unverändert lassen
		}
		setter := &mq.MailBlacklistSetter{
			Email:  omit.From(strings.TrimSpace(ent.sup.EmailAddress)),
			Reason: omit.From(ent.sup.SuppressionReason),
			Origin: omit.From(ent.sup.Origin),
			Stream: omit.From(ent.stream),
			Source: omit.From("postmark"),
		}
		if t, perr := time.Parse(time.RFC3339, ent.sup.CreatedAt); perr == nil {
			setter.CreatedAt = omit.From(t)
		}
		// ON CONFLICT (lower(email)) DO NOTHING macht den Insert idempotent: ein
		// paralleler Sync (Cron-Lauf ⟷ "Jetzt synchronisieren"-Button) oder ein
		// bereits vorhandener Eintrag kollidiert nicht mehr mit dem unique-Index,
		// sondern wird übersprungen. Bei einem Konflikt liefert RETURNING keine
		// Zeile → sql.ErrNoRows; das ist kein Fehler, sondern schlicht
		// "schon gesperrt".
		_, insErr := mq.MailBlacklists.Insert(setter,
			im.OnConflict(psql.Raw("lower(email)")).DoNothing(),
		).One(ctx, db.DBob)
		if insErr != nil {
			if errors.Is(insErr, sql.ErrNoRows) {
				continue
			}
			slog.Error("blacklist sync: insert", "err", insErr, "email", ent.sup.EmailAddress)
			continue
		}
		added++
	}

	// (4) reaktivierte postmark-Zeilen entfernen.
	removed := 0
	for key, e := range byEmail {
		if e.Source != "postmark" {
			continue // manuelle Einträge bleiben
		}
		if _, ok := current[key]; ok {
			continue // weiterhin gesperrt
		}
		if delErr := e.Delete(ctx, db.DBob); delErr != nil {
			slog.Error("blacklist sync: delete", "err", delErr, "email", e.Email)
			continue
		}
		removed++
	}

	res := &SyncResult{Added: added, Removed: removed, Total: len(current)}
	slog.Info("blacklist sync done", "added", res.Added, "removed", res.Removed,
		"total", res.Total, "streams", strings.Join(streams, ","))
	return res, nil
}

// activeStreamIDs liefert die eindeutigen, nicht-leeren Postmark-Stream-IDs aller
// aktiven Absender-Domains.
func activeStreamIDs(ctx context.Context) ([]string, error) {
	domains, err := mq.MailDomains.Query(
		sm.Where(mq.MailDomains.Columns.IsActive.EQ(psql.Arg(true))),
	).All(ctx, db.DBob)
	if err != nil {
		return nil, fmt.Errorf("load mail_domain: %w", err)
	}
	seen := make(map[string]struct{})
	var streams []string
	for _, d := range domains {
		s := strings.TrimSpace(d.PostmarkStreamID)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		streams = append(streams, s)
	}
	return streams, nil
}

// RemoveFromBlacklist nimmt eine einzelne Adresse wieder aus der Blacklist. Sie
// wird zuerst in Postmark reaktiviert (Suppression gelöscht) — sonst würde der
// nächste Sync sie sofort erneut aufnehmen — und erst danach lokal entfernt.
// Lässt Postmark die Reaktivierung nicht zu (z. B. SpamComplaint), bleibt der
// lokale Eintrag bestehen und es wird ein Fehler mit der Postmark-Begründung
// zurückgegeben.
func RemoveFromBlacklist(ctx context.Context, id int32) error {
	row, err := mq.FindMailBlacklist(ctx, db.DBob, id)
	if err != nil || row == nil {
		return errors.New("Blacklist-Eintrag nicht gefunden")
	}

	// In Postmark reaktivieren, falls ein Stream hinterlegt ist (gespiegelte
	// Suppression). Rein manuelle Einträge ohne Stream werden nur lokal entfernt.
	if strings.TrimSpace(row.Stream) != "" {
		if conf.C.Postmark.ServerToken == "" {
			return errors.New("postmark: ServerToken nicht konfiguriert")
		}
		client := NewClient(conf.C.Postmark.ServerToken)
		results, derr := client.DeleteSuppressions(ctx, row.Stream, []string{row.Email})
		if derr != nil {
			return fmt.Errorf("Postmark-Reaktivierung fehlgeschlagen: %w", derr)
		}
		for _, r := range results {
			if strings.EqualFold(r.Status, "Failed") {
				msg := strings.TrimSpace(r.Message)
				if msg == "" {
					msg = "Adresse kann in Postmark nicht reaktiviert werden (z. B. SpamComplaint)"
				}
				return errors.New(msg)
			}
		}
	}

	if err := row.Delete(ctx, db.DBob); err != nil {
		return fmt.Errorf("delete blacklist row: %w", err)
	}
	slog.Info("blacklist: address removed", "email", row.Email, "stream", row.Stream)
	return nil
}

// IsBlacklisted prüft, ob eine einzelne Adresse in der globalen Blacklist steht.
func IsBlacklisted(ctx context.Context, email string) (bool, error) {
	key := normalizeEmail(email)
	if key == "" {
		return false, nil
	}
	return mq.MailBlacklists.Query(
		sm.Where(psql.Raw("lower(email) = ?", key)),
	).Exists(ctx, db.DBob)
}

// LoadBlacklistSet lädt alle geblockten Adressen als Set (lower(email)) für
// effizientes Vorfiltern größerer Empfängerlisten.
func LoadBlacklistSet(ctx context.Context) (map[string]struct{}, error) {
	rows, err := mq.MailBlacklists.Query().All(ctx, db.DBob)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		set[normalizeEmail(r.Email)] = struct{}{}
	}
	return set, nil
}

// normalizeEmail trimmt und lowercased eine Adresse für den Blacklist-Abgleich
// (deckt sich mit dem unique-Index lower(email)).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
