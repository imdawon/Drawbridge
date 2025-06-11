package ui

import (
	"imdawon/drawbridge/cmd/dashboard/ui/templates"
	"imdawon/drawbridge/cmd/drawbridge/emissary"
	"log/slog"
	"net/http"
)

// Router interface for route registration
type Router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
	Patch(pattern string, handlerFn http.HandlerFunc)
}

// Add statistics endpoints to the controller
func (f *Controller) RegisterStatisticsEndpoints(r Router) {
	r.Get("/stats/clients", f.handleStatsClients)
	r.Get("/stats/services", f.handleStatsServices)
	r.Get("/stats/connections", f.handleStatsConnections)
	r.Get("/stats/events", f.handleStatsEvents)
}

// Handler for client statistics
func (f *Controller) handleStatsClients(w http.ResponseWriter, r *http.Request) {
	clients, err := f.DB.GetAllEmissaryClients()
	if err != nil {
		slog.Error("error getting client statistics", slog.Any("error", err))
	}

	totalClients := len(clients)
	activeClients := 0
	revokedClients := 0

	for _, client := range clients {
		if client.Revoked == 0 {
			activeClients++
		} else {
			revokedClients++
		}
	}

	templates.GetStatsClients(totalClients, activeClients, revokedClients).Render(r.Context(), w)
}

// Handler for service statistics
func (f *Controller) handleStatsServices(w http.ResponseWriter, r *http.Request) {
	services, err := f.DB.GetAllServices()
	if err != nil {
		slog.Error("error getting service statistics", slog.Any("error", err))
	}

	totalServices := len(services)
	topService := "None"
	topServiceConnections := 0

	if totalServices > 0 {
		// In a full implementation we would count actual connections
		// For now just use the first service
		topService = services[0].Name
		topServiceConnections = 0
	}

	templates.GetStatsServices(services, totalServices, topService, topServiceConnections).Render(r.Context(), w)
}

// Handler for connection statistics
func (f *Controller) handleStatsConnections(w http.ResponseWriter, r *http.Request) {
	// In a full implementation we would count connections from the database
	totalConnections := 0
	todayConnections := 0

	// Example connection types - would be from database in full implementation
	connectionTypes := []templates.ConnectionStat{
		{Name: "HTTP", Count: 15},
		{Name: "SSH", Count: 8},
		{Name: "TCP", Count: 12},
	}

	templates.GetStatsConnections(totalConnections, todayConnections, connectionTypes).Render(r.Context(), w)
}

// Handler for event statistics
func (f *Controller) handleStatsEvents(w http.ResponseWriter, r *http.Request) {
	// Get recent events - limit to last 10
	clients, err := f.DB.GetAllEmissaryClients()
	if err != nil {
		slog.Error("error getting client events", slog.Any("error", err))
	}

	events := make([]emissary.Event, 0)

	// Only try to get events if we have clients
	if len(clients) > 0 {
		var deviceIDs []any
		for _, client := range clients {
			deviceIDs = append(deviceIDs, client.ID)
		}

		latestClientEvents, err := f.DB.GetLatestEventForEachDeviceId(deviceIDs)
		if err != nil {
			slog.Error("error getting latest client events", slog.Any("error", err))
		} else {
			// Convert map to slice for template
			for _, event := range latestClientEvents {
				events = append(events, event)
			}
		}
	}

	templates.GetStatsEvents(events).Render(r.Context(), w)
}
