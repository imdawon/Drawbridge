package main

import (
	"dhens/drawbridge/cmd/dashboard/ui"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	"dhens/drawbridge/cmd/drawbridge/services"
	flagger "dhens/drawbridge/cmd/flags"
	"dhens/drawbridge/cmd/utils"
	"flag"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

func main() {
	flagger.FLAGS = &flagger.CommandLineArgs{}
	flag.StringVar(
		&flagger.FLAGS.FrontendAPIHostAndPort,
		"fapi",
		"localhost:3000",
		"listening host and port for frontend api e.g localhost:3000",
	)
	flag.StringVar(
		&flagger.FLAGS.BackendAPIHostAndPort,
		"api",
		"localhost:3001",
		"listening host and port for backend api e.g localhost:3001",
	)
	flag.StringVar(
		&flagger.FLAGS.SqliteFilename,
		"sqlfile",
		"drawbridge.db",
		"file name for Drawbridge sqlite database",
	)
	flag.StringVar(
		&flagger.FLAGS.Env,
		"env",
		"production",
		"the environment that Drawbridge is running in (production, development)",
	)
	flag.Parse()

	// Show debugger messages in development mode.
	if flagger.FLAGS.Env == "development" {
		programLevel := new(slog.LevelVar)
		programLevel.Set(slog.LevelDebug)
		h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})
		slog.SetDefault(slog.New(h))
	}

	// Append Drawbridge binary location to sqlite filepath to avoid writing to home directory.
	// Ensure we are only reading files from our executable and not where the terminal is executing from.
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDirPath := path.Dir(execPath)
	flagger.FLAGS.SqliteFilename = filepath.Join(execDirPath, flagger.FLAGS.SqliteFilename)

	// Migrate sqlite tables
	db := persistence.NewSQLiteRepository(persistence.OpenDatabaseFile(flagger.FLAGS.SqliteFilename))
	err = db.MigrateServices()
	if err != nil {
		log.Fatalf("Error running services db migration: %s", err)
	}
	err = db.MigrateEmissaryClient()
	if err != nil {
		log.Fatalf("Error running emissary_client db migration: %s", err)
	}
	err = db.MigrateEmissaryClientEvent()
	if err != nil {
		log.Fatalf("Error running emissary_client_event db migration: %s", err)
	}
	err = db.MigrateDrawbridgeConfig()
	if err != nil {
		log.Fatalf("Error running drawbridge_config db migration: %s", err)
	}

	drawbridgeAPI := &drawbridge.Drawbridge{
		ProtectedServices: make(map[int64]services.RunningProtectedService, 0),
		DB:                db,
	}

	// Onboarding configuration has been complete and we can load all existing config files and start servers.
	// Otherwise, we set up the certificate authority and dependent servers once the user submits
	// their listening address via the onboarding popup modal, which POSTs to /admin/post/config.
	services, err := db.GetAllServices()
	if err != nil {
		log.Fatalf("Could not get all services: %s", err)
	}

	// Check if a listening address has been saved in either the old config/listening_address.txt file
	// or the database.
	listeningAddress, err := db.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		slog.Error("Database", slog.Any("Error: %s", err))
	}
	if listeningAddress == nil {
		if utils.FileExists("config/listening_address.txt") {
			addressBytes := utils.ReadFile("config/listening_address.txt")
			if addressBytes != nil {
				address := string(*addressBytes)
				listeningAddress = &address
				err = db.CreateNewDrawbridgeConfigSettings("listening_address", *listeningAddress)
				if err != nil {
					slog.Error("Database Insert", slog.Any("error saving listening_address to drawbridge_config table", err))
				} else {
					// TODO
					// Make sure we are a good citizen and not deleting folders without user confirmation.
					// utils.DeleteDirectory("config")
				}
			}
		}
	} else {
		go drawbridgeAPI.SetUpCAAndDependentServices(services)
	}

	frontendController := ui.Controller{
		DrawbridgeAPI:     drawbridgeAPI,
		ProtectedServices: services,
		DB:                db,
	}

	// Set up templ controller used to return hypermedia to our htmx frontend.
	frontendController.SetUp(flagger.FLAGS.FrontendAPIHostAndPort)
}
