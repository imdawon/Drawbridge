package main

import (
	frontend "dhens/drawbridge/cmd/dashboard/ui"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/db"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"flag"
	"log"
)

type CommandLineArgs struct {
	frontendAPIHostAndPort string
	backendAPIHostAndPort  string
	sqliteFilename         string
}

func main() {
	flags := &CommandLineArgs{}
	flag.StringVar(
		&flags.frontendAPIHostAndPort,
		"fapi",
		"localhost:3000",
		"listening host and port for frontend api e.g localhost:3000",
	)
	flag.StringVar(
		&flags.backendAPIHostAndPort,
		"api",
		"localhost:3001",
		"listening host and port for backend api e.g localhost:3001",
	)
	flag.StringVar(
		&flags.sqliteFilename,
		"sqlfile",
		"dashboard.db",
		"file name for sqlite database",
	)
	flag.Parse()

	dbHandle := db.OpenDatabaseFile(flags.sqliteFilename)
	sqliteRepository := db.NewSQLiteRepository(dbHandle)
	err := sqliteRepository.Migrate()
	if err != nil {
		log.Fatalf("Error running db migration: %s", err)
	}
	frontendController := frontend.Controller{
		Sql: sqliteRepository,
	}

	ca := &certificates.CA{}
	err = ca.SetupCertificates()
	if err != nil {
		log.Fatalf("Error setting up root CA: %s", err)
	}

	// Set up templ controller used to return hypermedia to our htmx frontend.
	go func() {
		frontendController.SetUp(flags.frontendAPIHostAndPort, ca)
	}()

	// Set up mTLS http server
	go func() {
		drawbridge.SetUpReverseProxy(ca)
	}()

	drawbridge.SetUp(flags.backendAPIHostAndPort, ca)
}
