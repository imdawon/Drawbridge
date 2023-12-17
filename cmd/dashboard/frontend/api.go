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

	r.HandleFunc("/service/create", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newService := &backend.ProtectedService{}
		decoder.Decode(newService, r.Form)
		f.Sql.CreateNewService(*newService)
		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)
	})

	r.HandleFunc("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			f.handleGetService(w, r)
		default:
			log.Fatalf("%s is not permitted for this endpoint", r.Method)
		}
	})

	r.HandleFunc("/service/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "DELETE":
			f.handleDeleteService(w, r)
		default:
			log.Fatalf("%s is not permitted for this endpoint", r.Method)
		}
	})

	r.HandleFunc("/service/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			vars := mux.Vars(r)
			idString := vars["id"]
			id, err := strconv.Atoi(idString)
			if err != nil {
				log.Fatal("Error converting id string to int")
			}

			service, err := f.Sql.GetServiceById(int64(id))
			if err != nil {
				log.Fatalf("Could not get service: %s", err)
			}
			templates.EditServices(service).Render(r.Context(), w)
		case "PUT":
			f.handleEditService(w, r)
		default:
			log.Fatalf("%s is not permitted for this endpoint", r.Method)
		}
	})

	r.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)
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

func (f *Controller) handleGetService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idString := vars["id"]
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatal("Error converting id string to int")
	}

	service, err := f.Sql.GetServiceById(int64(id))
	if err != nil {
		log.Fatalf("Could not get service: %s", err)
	}
	templates.GetService(service).Render(r.Context(), w)
}

func (f *Controller) handleEditService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idString := vars["id"]
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatal("Error converting id string to int")
	}

	r.ParseForm()
	newService := &backend.ProtectedService{}
	decoder.Decode(newService, r.Form)

	err = f.Sql.UpdateService(newService, int64(id))
	if err != nil {
		log.Fatalf("Could not update service: %s", err)
	}
	services, err := f.Sql.GetAllServices()
	if err != nil {
		log.Fatalf("Could not get all services: %s", err)
	}
	templates.GetServices(services).Render(r.Context(), w)
}

func (f *Controller) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idString := vars["id"]
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatal("Error converting id string to int")
	}
	err = f.Sql.DeleteService(id)
	if err != nil {
		log.Fatalf("Could not delete service: %s", err)
	}
	services, err := f.Sql.GetAllServices()
	if err != nil {
		log.Fatalf("Could not get all services: %s", err)
	}
	templates.GetServices(services).Render(r.Context(), w)
}
