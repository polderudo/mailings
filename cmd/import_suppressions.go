package cmd

import (
	"app/mail/postmark"
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(importSuppressionsCmd)
}

// importSuppressionsCmd liest die aktuell in Postmark gesperrten Adressen
// (Suppression-Liste aller aktiven Absender-Domains) und spiegelt sie in die
// lokale mail_blacklist. Idempotent — kann beliebig oft ausgeführt werden und
// macht zugleich das, was der 15-min-Cron-Job laufend tut. Nutzung:
//
//	./app importSuppressions
var importSuppressionsCmd = &cobra.Command{
	Use:   "importSuppressions",
	Short: "Postmark-Suppressions (Bounces/Spam/Unsubscribes) in die Blacklist importieren",
	Long: `Ruft den Postmark Suppressions-Dump aller aktiven Absender-Domains ab und
gleicht ihn mit der lokalen mail_blacklist ab: neue Sperren werden aufgenommen,
in Postmark reaktivierte Adressen entfernt. Manuell gesetzte Einträge bleiben
unangetastet.`,
	Run: func(cmd *cobra.Command, args []string) {
		res, err := postmark.SyncSuppressions(context.Background())
		if err != nil {
			log.Fatalf("Suppression-Import fehlgeschlagen: %v", err)
		}
		log.Printf("Suppression-Import OK: %d hinzugefügt, %d entfernt, %d gesamt gesperrt",
			res.Added, res.Removed, res.Total)
		os.Exit(0)
	},
}
