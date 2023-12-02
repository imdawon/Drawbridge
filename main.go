package main

import (
	"dhens/drawbridge/cmd/dashboard/api"
	"dhens/drawbridge/cmd/dashboard/frontend"
	"flag"
)

func main() {
	var frontendAPIHostAndPort string
	flag.StringVar(
		&frontendAPIHostAndPort,
		"fapi",
		"localhost:3000",
		"listening host and port for frontend api e.g localhost:3000",
	)
	var backendAPIHostAndPort string
	flag.StringVar(
		&backendAPIHostAndPort,
		"api",
		"localhost:3000",
		"listening host and port for backend api e.g localhost:3001",
	)
	flag.Parse()

	go func() {
		frontend.SetUpFrontendAPIService(frontendAPIHostAndPort)
	}()

	api.SetUpGenericAPIService(backendAPIHostAndPort)

}
