package main

import (
	"dhens/drawbridge/cmd/analytics"
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
	"time"
)

func main() {
	flagger.FLAGS = &flagger.CommandLineArgs{}
	flag.StringVar(
		&flagger.FLAGS.FrontendAPIHostAndPort,
		"fapi",
		"localhost:3000",
		"listening host and port for the drawbridge dashboard page e.g localhost:3000",
	)
	flag.StringVar(
		&flagger.FLAGS.BackendAPIHostAndPort,
		"api",
		"localhost:3001",
		"(currently unused) listening host and port for emissary api e.g localhost:3001",
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
		"the environment that Drawbridge is running in (production, development). development mode increases logging verbosity.",
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
	if *listeningAddress == "" {
		if utils.FileExists("config/listening_address.txt") {
			addressBytes := utils.ReadFile("config/listening_address.txt")
			// If someone has a historical Drawbridge install, we will insert their listening address
			// into sqlite to get them up-to-date.
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

	// Initalize DAU ping only if enabled by the Drawbridge admin.
	dauPingEnabled, err := db.GetDrawbridgeConfigValueByName("dau_ping_enabled")
	if err != nil {
		slog.Error("Database", slog.Any("Error getting dau_ping_enabled: %s", err))
	} else if *dauPingEnabled == "true" {
		lastPingTime, err := db.GetDrawbridgeConfigValueByName("last_ping_timestamp")
		if err != nil {
			slog.Error("Database", slog.Any("Error getting last_ping_timestamp: %s", err))
		}
		// Parse timestamp if it exists and we didn't error out earlier.
		if *lastPingTime != "" && err == nil {
			lastPingTimestamp, err := time.Parse(time.RFC3339, *lastPingTime)
			if err != nil {
				slog.Error("Time Parse", slog.Any("Error parsing last_ping_timestamp: %s", err))
			}
			// Drawbridge hasn't been run within the last 24 hours since the last ping, so we
			// can run a DAU ping immediately.
			if time.Since(lastPingTimestamp) >= time.Hour*24 {
				go analytics.DAUPing(db)
				// We haven't waited 24 hours since our last DAU ping, so we need to schedule the future time
				// to do one.
			} else {
				nextTimeToPing := time.Until(lastPingTimestamp.AddDate(0, 0, 1))
				slog.Debug("DAU Ping", slog.Any("Next Ping Time", nextTimeToPing))
				time.AfterFunc(nextTimeToPing, func() { analytics.DAUPing(db) })
			}
			// kick off DAU pings as it has been enabled but we can't get the latest ping timestamp.
		} else {
			analytics.DAUPing(db)
		}
	}

	frontendController := ui.Controller{
		DrawbridgeAPI:     drawbridgeAPI,
		ProtectedServices: services,
		DB:                db,
	}

	// Set up templ controller used to return hypermedia to our htmx frontend.
	frontendController.SetUp(flagger.FLAGS.FrontendAPIHostAndPort)

}
