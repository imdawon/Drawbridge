package frontend

import (
	"dhens/drawbridge/cmd/dashboard/backend"
	"dhens/drawbridge/cmd/dashboard/backend/db"
	"dhens/drawbridge/cmd/dashboard/frontend/templates"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
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

	r := mux.NewRouter()

	r.HandleFunc("/new-service", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newService := &backend.Service{}
		decoder.Decode(newService, r.Form)
		f.Sql.CreateNewService(*newService)
		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.Services(services).Render(r.Context(), w)
	})

	r.HandleFunc("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		if r.Method != "DELETE" {
			log.Fatalf("%s not permitted", r.Method)
		}
		idString := vars["id"]
		id, err := strconv.Atoi(idString)
		if err != nil {
			log.Fatal("Error converting id string to int")
		}
		err = f.Sql.DeleteService(id)
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		services, err := f.Sql.GetAllServices()
		templates.Services(services).Render(r.Context(), w)
	})

	r.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.Services(services).Render(r.Context(), w)
	})

	// FOr testing, so we can access the html files we create
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./cmd/dashboard/frontend/static")))

	srv := &http.Server{
		Handler: r,
		Addr:    hostAndPort,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
	return nil
}
