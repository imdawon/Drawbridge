package ui

import (
	"context"
	"dhens/drawbridge/cmd/dashboard/ui/templates"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	"dhens/drawbridge/cmd/drawbridge/types"
	flagger "dhens/drawbridge/cmd/flags"
	"dhens/drawbridge/cmd/utils"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
)

// Set a Decoder instance as a package global, because it caches
// meta-data about structs, and an instance can be shared safely.
var decoder = schema.NewDecoder()

type Controller struct {
	DrawbridgeAPI *drawbridge.Drawbridge
}

func (f *Controller) SetUp(hostAndPort string) error {
	slog.Info(fmt.Sprintf("Starting frontend api service on %s. Launching in default web browser...", hostAndPort))

	// Launch the Drawbridge Dashboard in the default browser.
	exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:3000").Start()
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

		// Now that have a listening address we can generate our certificate authority and start our other
		// services that require the CA to operate, like the mTLS reverse proxy.
		go f.DrawbridgeAPI.SetUpCAAndDependentServices()
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
		err := f.DrawbridgeAPI.CreateEmissaryClientTCPMutualTLSKey("testing")
		if err != nil {
			fmt.Fprintf(w, "Error saving Emissary Certificates and Key to local filesystem.")
		}
		fmt.Fprintf(w, "Successfully saved Emissary Certificates and Key to \"emissary_certs_and_key_here\" to local filesystem.")
	})

	r.Post("/service/create", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		newService := &types.ProtectedService{}
		decoder.Decode(newService, r.Form)
		newService, err := persistence.Services.CreateNewService(*newService)
		if err != nil {
			slog.Error("error creatng new service: %w", err)
		}

		services, err := persistence.Services.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)

		// Set up tcp reverse proxy that actually carries the client data to the desired service.
		ctx, cancel := context.WithCancel(context.Background())
		go f.DrawbridgeAPI.SetUpProtectedServiceTunnel(ctx, cancel, *newService)

	})

	r.Get("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.handleGetService(w, r)
	})

	r.Delete("/service/{id}/delete", f.handleDeleteService)

	r.Get("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		idString := chi.URLParam(r, "id")
		if idString == "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Unable to get service with a blank id")
		}
		id, err := strconv.Atoi(idString)
		if err != nil {
			log.Fatalf("Error converting idString %s to int %d: %s", idString, id, err)
		}

		service, err := persistence.Services.GetServiceById(int64(id))
		if err != nil {
			log.Fatalf("Could not get service: %s", err)
		}
		templates.EditServices(service).Render(r.Context(), w)
	})

	r.Get("/service/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		f.handleEditService(w, r)
	})

	r.Get("/services", func(w http.ResponseWriter, r *http.Request) {
		services, err := persistence.Services.GetAllServices()
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
	idString := chi.URLParam(r, "id")
	if idString == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Unable to get service with a blank id")
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatalf("Error converting idString %s to int %d: %s", idString, id, err)
	}

	service, err := persistence.Services.GetServiceById(int64(id))
	if err != nil {
		log.Fatalf("Could not get service: %s", err)
	}
	templates.GetService(service).Render(r.Context(), w)
}

func (f *Controller) handleEditService(w http.ResponseWriter, r *http.Request) {
	idString := chi.URLParam(r, "id")
	if idString == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Unable to edit service with a blank id")
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatalf("Error converting idString %s to int %d: %s", idString, id, err)
	}

	r.ParseForm()
	newService := &types.ProtectedService{}
	decoder.Decode(newService, r.Form)

	err = persistence.Services.UpdateService(newService, int64(id))
	if err != nil {
		log.Fatalf("Could not update service: %s", err)
	}
	services, err := persistence.Services.GetAllServices()
	if err != nil {
		log.Fatalf("Could not get all services: %s", err)
	}
	templates.GetServices(services).Render(r.Context(), w)
}

func (f *Controller) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	idString := chi.URLParam(r, "id")
	if idString == "" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Unable to delete service with a blank id")
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		log.Fatalf("Error converting idString %s to int %d: %s", idString, id, err)
	}
	err = persistence.Services.DeleteService(id)
	if err != nil {
		log.Fatalf("Could not delete service from database: %s", err)
		// TODO
		// render error deleting service template here.
	}
	err = f.DrawbridgeAPI.StopRunningProtectedService(int64(id))
	if err != nil {
		slog.Error(err.Error())
	}
	services, err := persistence.Services.GetAllServices()
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
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
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
