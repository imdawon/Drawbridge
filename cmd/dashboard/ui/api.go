package ui

import (
	"dhens/drawbridge/cmd/dashboard/ui/templates"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	flagger "dhens/drawbridge/cmd/flags"
	proxy "dhens/drawbridge/cmd/reverse_proxy"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"dhens/drawbridge/cmd/utils"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// Set a Decoder instance as a package global, because it caches
// meta-data about structs, and an instance can be shared safely.
var decoder = schema.NewDecoder()

type Controller struct {
	Sql *persistence.SQLiteRepository
	CA  *certificates.CA
}

func (f *Controller) SetUp(hostAndPort string, ca *certificates.CA) error {
	slog.Info(fmt.Sprintf("Starting frontend api service on %s", hostAndPort))

	services, err := f.Sql.GetAllServices()
	if err != nil {
		log.Fatalf("Could not get all services: %s", err)
	}
	// Start listener for all Protected Services
	for i, service := range services {
		// We only support 1 service at a time for now.
		// This will change once we manage our goroutines which run the tcp / udp proxy servers.
		if i > 1 {
			break
		}
		go proxy.SetUpProtectedServiceTunnel(service, ca)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/admin/get/config", func(w http.ResponseWriter, r *http.Request) {
		listeningAddressBytes := utils.ReadFile("config/listening_address.txt")
		if listeningAddressBytes != nil {
			listeningAddress := strings.TrimSpace(string(*listeningAddressBytes))
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, fmt.Sprintf("%s:%d", listeningAddress, 3100))
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "...")

		}
	})

	// Write the Drawbridge listener address to the listening_address.txt config file.
	// This file is read from when we create the certificates (certificate authority and server certificate for mTLS)
	// for Drawbridge.
	//
	// This endpoint will need to be refactored for two main reasons:
	// 1. Either support PATCH only or some other strategy.
	// In it's current form, it only updates one field (the listener address) and trashes the rest by way of making
	// an entirely new config.
	//
	// 2. We need to figure out how we want to store the drawbridge configuration.
	// The two most likely options that come to mind are going all in on sqlite for all data persistence,
	// or writing to a file with some format like JSON.
	r.Post("/admin/post/config", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newSettings := &drawbridge.Settings{}
		decoder.Decode(newSettings, r.Form)

		if newSettings.ListenerAddress == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "<span class=\"error-response\">Listening address is blank! Please try again.<span>")
		}

		err := utils.SaveFile("listening_address.txt", newSettings.ListenerAddress, "config")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "<span class=\"error-response\">Error saving listening address. Please try again.<span>")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, fmt.Sprintf("%s:%d", newSettings.ListenerAddress, 3100))
	})

	r.Get("/admin/get/onboarding_modal", func(w http.ResponseWriter, r *http.Request) {
		listeningAddressBytes := utils.ReadFile("config/listening_address.txt")
		if listeningAddressBytes == nil {
			templates.GetOnboardingModal().Render(r.Context(), w)
		} else {
			// Serve nothing since we already have set a listening address (onboarding has already happened).
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "")
		}
	})

	r.Get("/admin/get/emissary_auth_files", func(w http.ResponseWriter, r *http.Request) {
		err := f.CA.CreateEmissaryClientTCPMutualTLSKey("testing")
		if err != nil {
			fmt.Fprintf(w, "Error saving Emissary Certificates and Key to local filesystem.")
		}
		fmt.Fprintf(w, "Successfully saved Emissary Certificates and Key to \"emissary_certs_and_key_here\" to local filesystem.")
	})

	r.Post("/service/create", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newService := &drawbridge.ProtectedService{}
		decoder.Decode(newService, r.Form)
		newService, err := f.Sql.CreateNewService(*newService)
		if err != nil {
			slog.Error("error creatng new service: %w", err)
		}

		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)

		// Set up tcp reverse proxy that actually carries the client data to the desired service.
		go proxy.SetUpProtectedServiceTunnel(*newService, ca)
	})

	r.Get("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.handleGetService(w, r)
	})

	r.Delete("/service/{id}/delete", f.handleDeleteService)

	r.Get("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	})

	r.Get("/service/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		f.handleEditService(w, r)
	})

	r.Get("/services", func(w http.ResponseWriter, r *http.Request) {
		services, err := f.Sql.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)
	})

	// FOr testing, so we can access the html files we create
	workDir, _ := os.Getwd()
	if flagger.FLAGS.Env == "development" {
		filesDir := http.Dir(filepath.Join(workDir, "./cmd/dashboard/ui/static"))
		FileServer(r, "/", filesDir)
	} else {
		filesDir := http.Dir(filepath.Join(workDir, "./ui/static"))
		FileServer(r, "/", filesDir)
	}

	// Serve the ui / api.
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
	newService := &drawbridge.ProtectedService{}
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

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
