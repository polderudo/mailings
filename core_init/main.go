package core_init

import (
	"app/auth"
	coreConf "app/conf"
	"app/db"
	"app/ilog"
	"app/templates"
	"log"
)

func LoadConfig(configDir string) error {
	if err := coreConf.LoadConfig(configDir); err != nil {
		log.Fatalf("Could not load config file: %v\n", err)
	}

	if err := ilog.InitLog(); err != nil {
		log.Fatalf("Could not init log: %v\n", err)
	}

	return nil
}

// InitializeCore initializes the whole core system
// Further components that need initialization and are also used in external applications, should also be initialized here
func InitializeCore(rbacModel string) error {
	templates.InitTemplates()
	log.Println("core initialized")

	if err := auth.InitAuth(db.StdDB, rbacModel); err != nil {
		log.Fatalf("Error initializing Auth: %v\n", err)
	}

	return nil
}
