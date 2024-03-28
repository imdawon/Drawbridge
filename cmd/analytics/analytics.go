package analytics

import (
	"crypto/tls"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Send daily anonymized ping to measure a rough user count.
// Drawbridge will send a different ping value each day.
func DAUPing(db *persistence.SQLiteRepository) {
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	pingValue := rand.Int()
	pingMessage := fmt.Sprintf(`{
		"ping": %d	
	}`, pingValue)

	resp, err := client.Post("https://little-union-a5d0.dawsondev.workers.dev", "application/json", strings.NewReader(pingMessage))
	if err != nil {
		slog.Error("DAU Ping", slog.Any("Error", err))
	}

	if resp != nil {
		slog.Debug("DAU Ping", slog.String("Upload Result", resp.Status))
	} else {
		slog.Error("DAU Ping - Upload Failed - No Response")
	}
	newPingTimestamp := time.Now()
	newPingTime := newPingTimestamp.Format(time.RFC3339)
	err = db.CreateNewDrawbridgeConfigSettings("last_ping_timestamp", newPingTime)
	if err != nil {
		slog.Error("Database", slog.Any("Error setting last_ping_timestamp: %s", err))
	}

	timeUntilNextPingTime := time.Until(newPingTimestamp.AddDate(0, 0, 1))
	slog.Debug("DAU Ping", slog.Any("Next Ping Time", timeUntilNextPingTime.String()))
	// Call dauPing (this function) again in 24 hours.
	time.AfterFunc(timeUntilNextPingTime, func() {
		DAUPing(db)
	})

}
