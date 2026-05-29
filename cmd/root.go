package cmd

import (
	"app/api"
	"app/auth"
	"log"
	"os"
	"strings"

	"app/core_init"
	"app/db"
	"app/internals"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "app",
		Short: "This is Templ 1.0",
		Long:  `Templ is a new app!`,
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("Templ is up. Starting API...")
			if err := api.StartCoreAPI(); err != nil {
				log.Fatalf("Error Templ Starting API: %v\n", err)
			}
			log.Println("Templ-API is down")
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	//we need to set the crypting key before we load the config
	internals.DefaultEncryptKey = "RbtUxdXdGdxHIJutuWInRI31ffsSORYy"

	if err := core_init.LoadConfig(""); err != nil {
		log.Fatalf("Error reading core config: %v\n", err)
	} else {
		log.Println("Core-config set")
	}

	log.Println("Connecting database...")
	if err := db.ConnectDB(); err != nil {
		log.Fatalf("Error creating DbTx connection: %v\n", err)
	}
	log.Println("Database connected")

	if err := core_init.InitializeCore(auth.RbacModelV2); err != nil {
		log.Fatalf("Error initializing core: %v\n", err)
	}

	isMigrate := false
	for _, arg := range os.Args {
		if strings.Contains(arg, "migrate") {
			isMigrate = true
		}
	}

	if !isMigrate {
	}
}
