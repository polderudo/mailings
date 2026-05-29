package db

import (
	"app/conf"
	"bufio"
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob"

	"github.com/nakami-lounge-GmbH/tools/helpers"

	coreDB "app/core/bob"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

var (
	StdDB      *sql.DB
	DBob       bob.DB
	DBTimeZone *time.Location
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var flags = flag.NewFlagSet("goose", flag.ExitOnError)

func SetDB(db *pgxpool.Pool) {
	DBob = bob.NewDB(stdlib.OpenDBFromPool(db))
	StdDB = stdlib.OpenDBFromPool(db)
}

func loadDBLocation(name string, abbrev string) (*time.Location, error) {
	if name != "" {
		location, err := time.LoadLocation(name)
		if err == nil {
			return location, nil
		}
	}

	if abbrev != "" {
		location, err := time.LoadLocation(abbrev)
		if err == nil {
			return location, nil
		}
	}

	return nil, fmt.Errorf("unknown time zone name=%q abbrev=%q", name, abbrev)
}

// ConnectDB create a new DB connection
func ConnectDB() error {
	var err error

	var psqlInfo string
	if conf.C.DB.Password != "" {
		psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", conf.C.DB.Host, conf.C.DB.Port, conf.C.DB.User, conf.C.DB.Password, conf.C.DB.Schema)
	} else {
		psqlInfo = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable", conf.C.DB.Host, conf.C.DB.Port, conf.C.DB.User, conf.C.DB.Schema)
	}

	cnf, err := pgxpool.ParseConfig(psqlInfo)
	//cnf.MinConns = 1
	cnf.MaxConns = 100

	if err != nil {
		return fmt.Errorf("error parsing db config:%w", err)
	}

	ctx := context.Background()
	pxDB, err := pgxpool.NewWithConfig(ctx, cnf)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	//try to ping the db
	err = pxDB.Ping(ctx)
	if err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	SetDB(pxDB)

	//read the db time zone
	tz, err := coreDB.GetTimeZone().One(ctx, DBob)
	if err != nil {
		return fmt.Errorf("error getting db time zone: %w", err)
	}
	DBTimeZone, err = loadDBLocation(tz.Name, tz.Abbrev)
	if err != nil {
		return fmt.Errorf("error loading db time zone: %w", err)
	}

	return nil
}

// MigrateDB migrates the database to the specified version
// migrate must be the only param for the binary
// only goose params may be attached to it
func MigrateDB(up bool) error {
	var command string
	var arguments []string

	goose.SetBaseFS(migrationFiles)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose select-dialect: %v", err)
		os.Exit(1)
	}

	//connect DbTx
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", conf.C.DB.Host, conf.C.DB.Port, conf.C.DB.User, conf.C.DB.Password, conf.C.DB.Schema)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	//try to ping the db
	err = db.Ping()
	if err != nil {
		return fmt.Errorf("error pinging database: %w", err)
	}

	if !up {
		//check if goose called
		posCommandOff := 0
		if len(os.Args) == 1 || (len(os.Args) > 2 && (os.Args[2+posCommandOff] != "migrate") && os.Args[2+posCommandOff] != "migrateAll") {
			return nil
		}

		flags.Usage = usage
		if err := flags.Parse(os.Args[3+posCommandOff:]); err != nil {
			panic(err)
		}

		args := flags.Args()

		if len(args) == 0 {
			flags.Usage()
			os.Exit(0)
		}

		//if migrating down, and not silent mode, we ask user if he really wants to downgrade
		if !conf.C.IsDevSystem && (helpers.StringInSliceI("down", args) || helpers.StringInSliceI("down-to", args)) && !helpers.StringInSliceI("silent-down", args) {
			var err error
			reader := bufio.NewReader(os.Stdin)
			log.Print("Do you really want to downgrade? [y/n]: ")
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Panicln(err)
			}
			text = strings.Replace(text, "\n", "", -1)
			text = strings.Replace(text, "\r", "", -1)
			if text != "y" && text != "Y" {
				log.Println("Stopping (down)migration")
				os.Exit(1)
			}
		}

		command = args[0]
		if len(args) > 2 && command == "create" {
			if err := goose.Run("create", db, "migrations", args[1:]...); err != nil {
				log.Fatalf("goose run: %v", err)
			}
			log.Println("Created migration")
			os.Exit(0)
		}

		if len(args) < 1 {
			flags.Usage()
			os.Exit(1)
		}

		if command == "-h" || command == "--help" {
			flags.Usage()
			os.Exit(0)
		}

		if len(args) > 1 {
			arguments = append(arguments, args[1:]...)
		}
	} else {
		command = "up"
	}

	log.Println("Starting migration - Core")

	goose.SetTableName("goose_version")
	if err := goose.Run(command, db, "migrations", arguments...); err != nil {
		log.Fatalf("goose run: %v", err)
	}

	log.Println("DbTx-Migration - Core done")

	return nil
}

func usage() {
	log.Print(usageStr)
}

var (
	usageStr = `Usage: migrate COMMAND
Commands:
    up                                Migrate the DbTx to the most recent version available
    up-to VERSION                     Migrate the DbTx to a specific VERSION
    down [silent-down]                Roll back the version by 1 (silent-down without confirmation)
    down-to VERSION [silent-down]     Roll back to a specific VERSION (silent-down without confirmation)
    redo                              Re-run the latest migration
    status                            Dump the migration status for the current DbTx
    version                           Print the current version of the database
    create NAME [sql|go]              Creates new migration file with next version
`
)
