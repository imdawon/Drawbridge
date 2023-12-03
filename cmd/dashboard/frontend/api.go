package frontend

import (
	"dhens/drawbridge/cmd/dashboard/frontend/templates"
	"log"
	"net/http"
)

func SetUpAPI(hostAndPort string) {
	log.Printf("Starting frontend api service on %s", hostAndPort)
	// FOr testing, so we can access the html files we create
	http.Handle("/", http.FileServer(http.Dir("./cmd/dashboard")))

	http.HandleFunc("/new-service", func(w http.ResponseWriter, r *http.Request) {
		templates.Hello(r).Render(r.Context(), w)
	})

	log.Fatal(http.ListenAndServe(hostAndPort, nil))
}

func handleCreateNewService(r *http.Request) {

}
