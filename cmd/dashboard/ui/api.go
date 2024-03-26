package ui

import (
	"dhens/drawbridge/cmd/dashboard/ui/templates"
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	flagger "dhens/drawbridge/cmd/flags"
	"dhens/drawbridge/cmd/utils"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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
	DrawbridgeAPI     *drawbridge.Drawbridge
	ProtectedServices []drawbridge.ProtectedService
}

func (f *Controller) SetUp(hostAndPort string) error {
	slog.Info(fmt.Sprintf("Starting frontend api service on http://%s. Launching in default web browser...", hostAndPort))

	// Launch the Drawbridge Dashboard in the default browser.
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://localhost:3000").Start()
	case "darwin":
		exec.Command("open", "http://localhost:3000").Start()
	default:
		slog.Info("platform not supported for opening Drawbridge in default browser:", runtime.GOOS)
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/admin/get/emissary/bundle", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		emissaryBundleConfig := drawbridge.EmissaryConfig{}
		decoder.Decode(&emissaryBundleConfig, r.Form)

		bundledFile, err := f.DrawbridgeAPI.GenerateEmissaryBundle(emissaryBundleConfig)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `<span class="error">Error creating Emissary Bundle: %s. Please go back and try again.</span>`, err)
			return
		}
		fileContents := bundledFile.Contents
		w.WriteHeader(http.StatusOK)
		// Set the appropriate headers for the file download
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, bundledFile.Name))
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", string(len(*fileContents)))

		// Write the file bytes to the HTTP response
		_, err = w.Write(*fileContents)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	})

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
		} else if newSettings.ListenerAddress == "localhost" {
			newSettings.ListenerAddress = "127.0.0.1"
		}

		err := utils.SaveFile("listening_address.txt", newSettings.ListenerAddress, "config")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "<span class=\"error-response\">Error saving listening address. Please try again.<span>")
		}

		// Now that have a listening address we can generate our certificate authority and start our other
		// services that require the CA to operate, like the mTLS reverse proxy.
		go f.DrawbridgeAPI.SetUpCAAndDependentServices(f.ProtectedServices)
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
		newService := drawbridge.ProtectedService{}
		decoder.Decode(&newService, r.Form)
		// Rewrite a host input from the Drawbridge admin so we don't get errors trying to parse
		// the "localhost" string as a net.IP.
		if newService.Host == "localhost" {
			newService.Host = "127.0.0.1"
		}
		newServiceWithId, err := persistence.Drawbridge.CreateNewService(newService)
		if err != nil {
			slog.Error("error creatng new service: %w", err)
		}

		services, err := persistence.Drawbridge.GetAllServices()
		if err != nil {
			log.Fatalf("Could not get all services: %s", err)
		}
		templates.GetServices(services).Render(r.Context(), w)

		// Set up tcp reverse proxy that actually carries the client data to the target service.
		go f.DrawbridgeAPI.AddNewProtectedService(*newServiceWithId)

	})

	r.Get("/service/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.handleGetService(w, r)
	})

	r.Delete("/service/{id}/delete", f.handleDeleteService)

	r.Get("/service/{id}/edit", func(w http.ResponseWriter, r *http.Request) {
		idString := chi.URLParam(r, "id")
		if idString == "" {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Unable to get service with a blank id")
		}
		id, err := strconv.Atoi(idString)
		if err != nil {
			log.Fatalf("Error converting idString %s to int %d: %s", idString, id, err)
		}

		service, err := persistence.Drawbridge.GetServiceById(int64(id))
		if err != nil {
			log.Fatalf("Could not get service: %s", err)
		}
		templates.EditService(service).Render(r.Context(), w)
	})

	r.Patch("/service/{id}/edit", f.handleEditService)

	r.Get("/services", func(w http.ResponseWriter, r *http.Request) {
		services, err := persistence.Drawbridge.GetAllServices()
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
		ex, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		dir := path.Dir(ex)
		filesDir := http.Dir(fmt.Sprintf("%s/static", dir))
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

	service, err := persistence.Drawbridge.GetServiceById(int64(id))
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
	newService := &drawbridge.ProtectedService{}
	decoder.Decode(newService, r.Form)
	newService.ID = int64(id)

	err = persistence.Drawbridge.UpdateService(newService, int64(id))
	if err != nil {
		log.Fatalf("Could not update service: %s", err)
	}

	go f.DrawbridgeAPI.AddNewProtectedService(*newService)
	if err != nil {
		log.Fatalf("Failed to start Protected Service after it was edited by a Drawbridge admin: %s", err)
	}
	services, err := persistence.Drawbridge.GetAllServices()
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
	err = persistence.Drawbridge.DeleteService(id)
	if err != nil {
		log.Fatalf("Could not delete service from database: %s", err)
		// TODO
		// render error deleting service template here.
	}
	f.DrawbridgeAPI.StopRunningProtectedService(int64(id))
	if err != nil {
		slog.Error(err.Error())
	}
	services, err := persistence.Drawbridge.GetAllServices()
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
