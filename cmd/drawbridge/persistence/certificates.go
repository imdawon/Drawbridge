package persistence

import (
	"context"
	"database/sql"
)

func (r *SQLiteRepository) MigrateCertificates() error {
	query := `
	CREATE TABLE IF NOT EXISTS certificates(
		id TEXT NOT NULL UNIQUE,
		certificate TEXT NOT NULL,
		certificate_type TEXT NOT NULL,
		FOREIGN KEY(emissary_client_id) REFERENCES emissary_client(id),
		FOREIGN KEY(service_id) REFERENCES services(id),
		revoked INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_certificates_revoked ON certificates (revoked);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) RevokeEmissaryClientCertificate(clientID string) error {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil

}
