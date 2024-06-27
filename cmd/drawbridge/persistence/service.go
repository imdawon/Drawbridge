package persistence

import (
	"database/sql"
	"dhens/drawbridge/cmd/drawbridge/services"
	"fmt"
	"log"
	"log/slog"

	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

func (r *SQLiteRepository) MigrateServices() error {
	query := `
	CREATE TABLE IF NOT EXISTS services(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		host TEXT NOT NULL,
		port INTEGER NOT NULL
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewService(service services.ProtectedService) (*services.ProtectedService, error) {
	res, err := r.db.Exec(
		"INSERT INTO services(name, description, host, port) values(?,?,?,?)",
		service.Name,
		service.Description,
		service.Host,
		service.Port,
	)
	if err != nil {
		return nil, fmt.Errorf("error inserting new service into the db: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("a new service was inserted into the db, but an error was returned when retrieving the id for it: %w", err)
	}

	service.ID = id
	return &service, nil

}

func (r *SQLiteRepository) GetAllServices() ([]services.ProtectedService, error) {
	rows, err := r.db.Query("SELECT * from services")
	if err != nil {
		return nil, fmt.Errorf("error getting all services: %w", err)
	}
	defer rows.Close()

	var all []services.ProtectedService
	for rows.Next() {
		var service services.ProtectedService
		if err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Host,
			&service.Port); err != nil {
			return nil, err
		}
		all = append(all, service)
	}
	return all, nil
}

func (r *SQLiteRepository) GetServiceById(id int64) (*services.ProtectedService, error) {
	rows, err := r.db.Query("SELECT * FROM services WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("error getting service id %d: %s", id, err)
	}
	defer rows.Close()

	var service services.ProtectedService
	for rows.Next() {
		if err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Host,
			&service.Port); err != nil {
			return nil, err
		}
	}
	return &service, nil
}

func (r *SQLiteRepository) UpdateService(updated *services.ProtectedService, id int64) error {
	if id == 0 {
		return fmt.Errorf("invalid updated ID")
	}
	res, err := r.db.Exec(
		"UPDATE services SET name = ?, description = ?, host = ?, port = ? WHERE id = ?",
		updated.Name,
		updated.Description,
		updated.Host,
		updated.Port,
		id,
	)
	if err != nil {
		return fmt.Errorf("error updating service id %d: %s", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("update failed: %s", err)
	}

	return nil

}

func (r *SQLiteRepository) DeleteService(id int) error {
	res, err := r.db.Exec("DELETE FROM services WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("error deleting service with id of %d: %s", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows were deleted for service id: %d", id)
	}
	return err
}

func OpenDatabaseFile(filename string) *sql.DB {
	db, err := sql.Open("sqlite", fmt.Sprintf("%s?cache=shared", filename))
	db.SetMaxOpenConns(1)
	if err != nil {
		log.Fatalf("Error opening sqlite db: %s", err)
	}
	slog.Info("Opened database file")
	return db
}
