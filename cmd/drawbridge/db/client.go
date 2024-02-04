package db

import (
	"dhens/drawbridge/cmd/drawbridge"
	"dhens/drawbridge/cmd/drawbridge/client"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClients() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_clients(
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

func (r *SQLiteRepository) CreateNewEmissaryClient(client drawbridge.ProtectedService) (*drawbridge.ProtectedService, error) {
	res, err := r.db.Exec(
		"INSERT INTO emissary_client(name, description, host, port) values(?,?,?,?)",
		client.Name,
		client.Description,
		client.Host,
		client.Port,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new emissary client: %s", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	client.ID = id
	return &client, nil

}

func (r *SQLiteRepository) GetEmissaryClientById(id int64) (*client.EmissaryClient, error) {
	rows, err := r.db.Query("SELECT * FROM emissary_clients WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("error getting emissary client id %d: %s", id, err)
	}
	defer rows.Close()

	var client client.EmissaryClient
	for rows.Next() {
		if err := rows.Scan(
			&client.ID,
			&client.OperatingSystemVersion,
			&client.LastSuccessfulConfigEvalResponse,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
	}
	return &client, nil
}

func (r *SQLiteRepository) UpdateEmissaryClient(updated *client.EmissaryClient, id int64) error {
	if id == 0 {
		return fmt.Errorf("the emissary client id supplied is invalid. unable to update emissary client row")
	}
	res, err := r.db.Exec(
		"UPDATE emissary_clients SET os_version = ?, last_successful_eval = ? WHERE id = ?",
		updated.OperatingSystemVersion,
		updated.LastSuccessfulConfigEvalResponse,
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
	res, err := r.db.Exec("DELETE FROM emissary_clients WHERE id = ?", id)
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
