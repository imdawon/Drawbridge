package persistence

import (
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClient() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_client(
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		host TEXT NOT NULL,
		port INTEGER NOT NULL
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewEmissaryClient(client emissary.EmissaryClient) (*emissary.EmissaryClient, error) {
	_, err := r.db.Exec(
		"INSERT INTO emissary_client(id, hostname, operating_system_version, last_successful_policy_evaluation) values(?,?,?)",
		client.ID,
		client.Hostname,
		client.OperatingSystemVersion,
		client.LastSuccessfulPolicyEvaluation,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new emissary client: %s", err)
	}

	return &client, nil

}

func (r *SQLiteRepository) GetEmissaryClientById(id int64) (*emissary.EmissaryClient, error) {
	rows, err := r.db.Query("SELECT * FROM emissary_client WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("error getting emissary client id %d: %s", id, err)
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

func (r *SQLiteRepository) UpdateEmissaryClient(updated *emissary.EmissaryClient, id int64) error {
	if id == 0 {
		return fmt.Errorf("the emissary client id supplied is invalid. unable to update emissary client row")
	}
	res, err := r.db.Exec(
		"UPDATE emissary_client SET os_version = ?, last_successful_eval = ? WHERE id = ?",
		updated.OperatingSystemVersion,
		updated.LastSuccessfulPolicyEvaluation,
		id,
	)
	if err != nil {
		return fmt.Errorf("error updating emissary client with id of %d: %s", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no emissary client rows updated: %s", err)
	}

	return nil

}

func (r *SQLiteRepository) DeleteEmissaryClient(id int) error {
	res, err := r.db.Exec("DELETE FROM emissary_client WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting emissary client with id of %d: %s", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to delete emissary client")
	}
	return err
}
