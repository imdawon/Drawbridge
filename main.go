package main

import (
	"dhens/drawbridge/cmd/dashboard/ui"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	flagger "dhens/drawbridge/cmd/flags"
	"dhens/drawbridge/cmd/utils"
	"flag"
	"log"
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

	persistence.Services = persistence.NewSQLiteRepository(persistence.OpenDatabaseFile(flagger.FLAGS.SqliteFilename))

	err := persistence.Services.Migrate()
	if err != nil {
		log.Fatalf("Error running db migration: %s", err)
	}

	drawbridgeAPI := &drawbridge.Drawbridge{
		ProtectedServices: make(map[int64]drawbridge.RunningProtectedService, 0),
	}

	// Onboarding configuration has been complete and we can load all existing config files and start servers.
	// Otherwise, we set up the certificate authority and dependent servers once the user submits
	// their listening address via the onboarding popup modal, which POSTs to /admin/post/config.
	if utils.FileExists("config/listening_address.txt") {
		go drawbridgeAPI.SetUpCAAndDependentServices()
	}

	frontendController := ui.Controller{
		DrawbridgeAPI: drawbridgeAPI,
	}

	// Set up templ controller used to return hypermedia to our htmx frontend.
	frontendController.SetUp(flagger.FLAGS.FrontendAPIHostAndPort)
}
