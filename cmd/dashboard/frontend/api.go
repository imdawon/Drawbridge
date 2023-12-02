package frontend

import (
	"dhens/drawbridge/cmd/dashboard/frontend/templates"
	"log"
	"net/http"

	"github.com/a-h/templ"
)

func SetUpAPI(hostAndPort string) {
	log.Printf("Starting frontend api service on %s", hostAndPort)
	http.Handle("/", http.FileServer(http.Dir("./static")))

	component := templates.Hello("John")
	http.Handle("/api/new-service", templ.Handler(component))

	log.Fatal(http.ListenAndServe(hostAndPort, nil))
}
