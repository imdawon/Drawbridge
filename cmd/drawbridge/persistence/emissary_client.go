package persistence

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"encoding/hex"
	"fmt"
)

func (r *SQLiteRepository) MigrateEmissaryClient() error {
	query := `
	CREATE TABLE IF NOT EXISTS emissary_client(
		id TEXT PRIMARY KEY NOT NULL,
		name TEXT UNIQUE NOT NULL,
		drawbridge_certificate TEXT UNIQUE NOT NULL,
		revoked INTEGER NOT NULL
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewEmissaryClient(client emissary.EmissaryClient) (*emissary.EmissaryClient, error) {
	_, err := r.db.Exec(
		"INSERT INTO emissary_client(id, name, drawbridge_certificate, revoked) values(?, ?, ?, ?)",
		client.ID,
		client.Name,
		client.DrawbridgeCertificate,
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
	for rows.Next() {
		var client emissary.EmissaryClient
		if err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.DrawbridgeCertificate,
			&client.Revoked,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

func (r *SQLiteRepository) GetAllEmissaryClientCertificates() (map[string]emissary.DeviceCertificate, error) {
	rows, err := r.db.Query("SELECT drawbridge_certificate, id, revoked FROM emissary_client")
	if err != nil {
		return nil, fmt.Errorf("error getting all emissary clients: %s", err)
	}
	defer rows.Close()

	deviceCerts := make(map[string]emissary.DeviceCertificate, 0)
	for rows.Next() {
		var client emissary.EmissaryClient
		if err := rows.Scan(
			&client.DrawbridgeCertificate,
			&client.ID,
			&client.Revoked,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
		// We hash the certificate value to not keep certs in memory
		// and to shorten the length of the cert as we store it in memory.
		certificateBytes := sha256.Sum256([]byte(client.DrawbridgeCertificate))
		shaCertificate := hex.EncodeToString(certificateBytes[:])
		deviceCerts[shaCertificate] = emissary.DeviceCertificate{
			DeviceID: client.ID,
			Revoked:  client.Revoked,
		}
	}
	return deviceCerts, nil
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
			&client.DrawbridgeCertificate,
			&client.Revoked,
		); err != nil {
			return nil, fmt.Errorf("error scanning emissary client database row into a emissary client struct: %s", err)
		}
	}
	return &client, nil
}

func (r *SQLiteRepository) UpdateEmissaryClient(updated *emissary.EmissaryClient, id int64, column string) error {
	if id == 0 {
		return fmt.Errorf("the emissary client id supplied is invalid. unable to update emissary client row")
	}
	res, err := r.db.Exec(
		"UPDATE emissary_client SET ? = ? WHERE id = ?",
		column,
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
func (r *SQLiteRepository) RevokeEmissaryClient(deviceID string) (*emissary.EmissaryClient, *emissary.Event, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	_, err = tx.Exec("UPDATE emissary_client SET revoked = 1 WHERE id = ?", deviceID)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("error unrevoking emissary client with id of %s: %s", deviceID, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	client, err := r.GetEmissaryClientById(deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting emissary client after revoking it: %w", err)
	}

	event, err := r.GetLatestEventForDeviceId(deviceID)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting latest event for revoked client: %w", err)
	}
	return client, event, nil
}

// Marks a device as unrevoked, which allows it to connect to Drawbridge after not be allowed to.
// We do this by removing the Emissary client certificate from the Drawbridge's Certificate Revocation List.
func (r *SQLiteRepository) UnRevokeEmissaryClient(id string) (*emissary.EmissaryClient, *emissary.Event, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}

	_, err = tx.Exec("UPDATE emissary_client SET revoked = 0 WHERE id = ?", id)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("error unrevoking emissary client with id of %s: %s", id, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	client, err := r.GetEmissaryClientById(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting emissary client after unrevoking it: %w", err)
	}

	event, err := r.GetLatestEventForDeviceId(id)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting latest event for revoked client: %w", err)
	}
	return client, event, nil
}
