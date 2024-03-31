package persistence

import (
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClientEvent() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_client_event(
		id TEXT PRIMARY KEY,
		device_id TEXT,
		type TEXT NOT NULL,
		target_service TEXT,
		connection_type TEXT,
		timestamp TEXT NOT NULL,
		FOREIGN KEY(device_id) REFERENCES emissary_client(id)
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) InsertEmissaryClientEvent(client emissary.EmissaryClient) (*emissary.EmissaryClient, error) {
	_, err := r.db.Exec(
		"INSERT INTO emissary_client_event(id, hostname, operating_system_version, last_successful_policy_evaluation) values(?,?,?)",
		client.ID,
		client.Name,
		client.OperatingSystemVersion,
		client.LastSuccessfulPolicyEvaluation,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new emissary client: %s", err)
	}

	return &client, nil
}

func (r *SQLiteRepository) GetRecentEventsByDeviceId(deviceID int64, recordCount int64) (*emissary.EmissaryClient, error) {
	rows, err := r.db.Query("SELECT * FROM emissary_client_event WHERE emissary_client_id = ? ORDER BY timestamp DESC LIMIT = ?", deviceID, recordCount)
	if err != nil {
		return nil, fmt.Errorf("error getting emissary client events for deviceID %d: %s", deviceID, err)
	}
	defer rows.Close()

	var client emissary.EmissaryClient
	for rows.Next() {
		if err := rows.Scan(
			&client.ID,
			&client.OperatingSystemVersion,
			&client.LastSuccessfulPolicyEvaluation,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
	}
	return &client, nil
}
