package persistence

import (
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"dhens/drawbridge/cmd/utils"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClientEvent() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_client_event(
		id TEXT PRIMARY KEY,
		device_id TEXT NOT NULL,
		device_ip TEXT NOT NULL,
		type TEXT NOT NULL,
		target_service TEXT,
		connection_type TEXT,
		timestamp TEXT NOT NULL,
		FOREIGN KEY(device_id) REFERENCES emissary_client(id)
	);
	CREATE INDEX IF NOT EXISTS idx_emissary_client_event_device_id ON emissary_client_event (device_id);
	`

	_, err := r.db.Exec(query)
	return err
}

var queryLatestDeviceEventForEachDevice = `
SELECT
	e.id AS event_id,
	e.device_id,
	e.device_ip,
	e.type,
	e.target_service,
	e.connection_type,
	e.timestamp
FROM
    (
        SELECT
			id,
            device_id,
			device_ip,
            type,
            target_service,
            connection_type,
            timestamp,
            ROW_NUMBER() OVER (PARTITION BY device_id ORDER BY timestamp DESC) AS rn
        FROM emissary_client_event
        WHERE device_id IN (?)
    ) e
    JOIN emissary_client c ON e.device_id = c.id
WHERE
    e.rn = 1;
	`

func (r *SQLiteRepository) InsertEmissaryClientEvent(event emissary.Event) error {
	_, err := r.db.Exec(
		"INSERT INTO emissary_client_event(id, device_id, device_ip, type, target_service, connection_type, timestamp) values(?,?,?,?,?,?,?)",
		&event.ID,
		&event.DeviceID,
		&event.ConnectionIP,
		&event.Type,
		&event.TargetService,
		&event.ConnectionType,
		&event.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("error inserting emissary event: %s", err)
	}

	return nil
}

// Gets the latest event for each device to use in the Device Fleet view in the dashboard.
// Returns a map with the key being the device id and value being the event itself.
func (r *SQLiteRepository) GetLatestEventForEachDeviceId(deviceIDs []string) (map[string]emissary.Event, error) {
	deviceIDsLen := len(deviceIDs)
	if deviceIDsLen == 0 {
		return nil, fmt.Errorf("no deviceIDs supplied to get latest event for each device id")
	}
	rows, err := r.db.Query(queryLatestDeviceEventForEachDevice, deviceIDs[0])
	if err != nil {
		return nil, fmt.Errorf("error getting latest event for each emissary client: %s", err)
	}
	defer rows.Close()

	var event emissary.Event
	events := make(map[string]emissary.Event, deviceIDsLen)
	for rows.Next() {
		if err := rows.Scan(
			&event.ID,
			&event.DeviceID,
			&event.ConnectionIP,
			&event.Type,
			&event.TargetService,
			&event.ConnectionType,
			&event.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
		event.Timestamp = utils.BeautifulTimeSince(event.Timestamp)
		events[event.DeviceID] = event
	}
	return events, nil
}

func (r *SQLiteRepository) GetLatestEventForDeviceId(deviceID string) (*emissary.Event, error) {
	if len(deviceID) == 0 {
		return nil, fmt.Errorf("no deviceIDs supplied to get latest event for each device id")
	}
	rows, err := r.db.Query(queryLatestDeviceEventForEachDevice, deviceID)
	if err != nil {
		return nil, fmt.Errorf("error getting latest event for each emissary client: %s", err)
	}
	defer rows.Close()

	var event emissary.Event
	for rows.Next() {
		if err := rows.Scan(
			&event.ID,
			&event.DeviceID,
			&event.ConnectionIP,
			&event.Type,
			&event.TargetService,
			&event.ConnectionType,
			&event.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
	}
	return &event, nil
}

// Used for one device. Mainly used to calculate averages or a range-based result.
// func (r *SQLiteRepository) GetRecentEventsByDeviceId(deviceID int64, recordCount int64) ([]emissary.Event, error) {
// 	rows, err := r.db.Query("SELECT * FROM emissary_client_event WHERE emissary_client_id = ? ORDER BY timestamp DESC LIMIT = ?", deviceID, recordCount)
// 	if err != nil {
// 		return nil, fmt.Errorf("error getting emissary client events for deviceID %d: %s", deviceID, err)
// 	}
// 	defer rows.Close()

// 	var client emissary.EmissaryClient
// 	for rows.Next() {
// 		if err := rows.Scan(
// 			&client.ID,
// 			&client.OperatingSystemVersion,
// 			&client.LastSuccessfulPolicyEvaluation,
// 		); err != nil {
// 			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
// 		}
// 	}
// 	return client, nil
// }
