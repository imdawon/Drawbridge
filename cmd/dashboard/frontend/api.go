package frontend

import (
	"dhens/drawbridge/cmd/dashboard/backend"
	"dhens/drawbridge/cmd/dashboard/backend/db"
	"dhens/drawbridge/cmd/dashboard/frontend/templates"
	"log"
	"net/http"

	"github.com/gorilla/schema"
)

// Set a Decoder instance as a package global, because it caches
// meta-data about structs, and an instance can be shared safely.
var decoder = schema.NewDecoder()

type Controller struct {
	Sql *db.SQLiteRepository
}

func (f *Controller) SetUp(hostAndPort string) error {
	log.Printf("Starting frontend api service on %s", hostAndPort)
	// FOr testing, so we can access the html files we create
	http.Handle("/", http.FileServer(http.Dir("./cmd/dashboard")))

	http.HandleFunc("/new-service", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newService := &backend.Service{}
		decoder.Decode(newService, r.Form)
		f.Sql.CreateNewService(*newService)
		templates.Hello(*newService).Render(r.Context(), w)
	})

	log.Fatal(http.ListenAndServe(hostAndPort, nil))
	return nil
}
