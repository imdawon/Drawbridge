package main

import (
	"dhens/drawbridge/cmd/dashboard/backend"
	"dhens/drawbridge/cmd/dashboard/frontend"
	"flag"
)

type ArgFlags struct {
	frontendAPIHostAndPort string
	backendAPIHostAndPort  string
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
		"localhost:3000",
		"listening host and port for backend api e.g localhost:3001",
	)
	flag.Parse()

	go func() {
		frontend.SetUpAPI(flags.frontendAPIHostAndPort)
	}()

	backend.SetUpAPI(flags.backendAPIHostAndPort)
}
