package main

import (
	"dhens/drawbridge/cmd/dashboard/backend"
	"dhens/drawbridge/cmd/dashboard/backend/db"
	"dhens/drawbridge/cmd/dashboard/frontend"
	"flag"
)

type ArgFlags struct {
	frontendAPIHostAndPort string
	backendAPIHostAndPort  string
	sqliteFilename         string
}

func main() {
	flags := &ArgFlags{}
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
	frontendController := frontend.Controller{
		Sql: sqliteRepository,
	}

	go func() {
		frontendController.SetUp(flags.frontendAPIHostAndPort)
	}()

	backend.SetUp(flags.backendAPIHostAndPort)
}
