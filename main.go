package main

import (
	frontend "dhens/drawbridge/cmd/dashboard/ui"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	flagger "dhens/drawbridge/cmd/flags"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
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

	dbHandle := persistence.OpenDatabaseFile(flagger.FLAGS.SqliteFilename)
	sqliteRepository := persistence.NewSQLiteRepository(dbHandle)
	err := sqliteRepository.Migrate()
	if err != nil {
		log.Fatalf("Error running db migration: %s", err)
	}

	ca := &certificates.CA{}
	err = ca.SetupCertificates()
	if err != nil {
		log.Fatalf("Error setting up root CA: %s", err)
	}

	frontendController := frontend.Controller{
		Sql: sqliteRepository,
		CA:  ca,
	}

	// Set up templ controller used to return hypermedia to our htmx frontend.
	go func() {
		frontendController.SetUp(flagger.FLAGS.FrontendAPIHostAndPort, ca)
	}()

	// Set up mTLS http server
	go func() {
		drawbridge.SetUpReverseProxy(ca)
	}()

	drawbridge.SetUpEmissaryAPI(flagger.FLAGS.BackendAPIHostAndPort, ca)

}
