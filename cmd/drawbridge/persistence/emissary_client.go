package persistence

import (
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClient() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_client(
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE,
		revoked INTEGER
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewEmissaryClient(client emissary.EmissaryClient) (*emissary.EmissaryClient, error) {
	_, err := r.db.Exec(
		"INSERT INTO emissary_client(id, name, revoked) values(?, ?, ?)",
		client.ID,
		client.Name,
		client.Revoked,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating new emissary client: %s", err)
	}

	return &client, nil

}

func (r *SQLiteRepository) GetAllEmissaryClients() ([]*emissary.EmissaryClient, error) {
	rows, err := r.db.Query("SELECT * FROM emissary_client")
	if err != nil {
		return nil, fmt.Errorf("error getting all emissary clients: %s", err)
	}
	defer rows.Close()

	var clients []*emissary.EmissaryClient
	var client emissary.EmissaryClient
	for rows.Next() {
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Revoked,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

func (r *SQLiteRepository) GetEmissaryClientById(id string) (*emissary.EmissaryClient, error) {
	rows, err := r.db.Query("SELECT * FROM emissary_client WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("error getting emissary client id %s: %s", id, err)
	}
	defer rows.Close()

	var client emissary.EmissaryClient
	for rows.Next() {
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Revoked,
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
		"UPDATE emissary_client SET name = ? WHERE id = ?",
		updated.Name,
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

// Marks a device as revoked, which keeps it from being able to connect to Drawbridge at all.
// We do this by adding the Emissary client certificate to Drawbridge's Certificate Revocation List.
func (r *SQLiteRepository) RevokeEmissaryClient(id string) (*emissary.EmissaryClient, *emissary.Event, error) {
	_, err := r.db.Exec("UPDATE emissary_client SET revoked = 1 WHERE id = ?", id)
	if err != nil {
		return nil, nil, fmt.Errorf("error unrevoking emissary client with id of %s: %s", id, err)

	}
	client, err := r.GetEmissaryClientById(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting emissary client after unrevoking it: %w", err)
	}

	event, err := r.GetLatestEventForDeviceId(client.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting latest event for revoked client: %w", err)
	}
	return client, event, nil

}

// Marks a device as unrevoked, which allows it to connect to Drawbridge after not be allowed to.
// We do this by removing the Emissary client certificate from the Drawbridge's Certificate Revocation List.
func (r *SQLiteRepository) UnRevokeEmissaryClient(id string) (*emissary.EmissaryClient, *emissary.Event, error) {
	_, err := r.db.Exec("UPDATE emissary_client SET revoked = 0 WHERE id = ?", id)
	if err != nil {
		return nil, nil, fmt.Errorf("error unrevoking emissary client with id of %s: %s", id, err)
	}

	client, err := r.GetEmissaryClientById(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting emissary client after unrevoking it: %w", err)
	}
	event, err := r.GetLatestEventForDeviceId(client.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting latest event for revoked client: %w", err)
	}
	return client, event, nil
}
