package db

import (
	"database/sql"
	"dhens/drawbridge/cmd/dashboard/backend"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{
		db: db,
	}
}

func (r *SQLiteRepository) Migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS services(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		host TEXT NOT NULL,
		port INTEGER NOT NULL
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewService(service backend.Service) (*backend.Service, error) {
	res, err := r.db.Exec(
		"INSERT INTO services(name, description, host, port) values(?,?,?,?)",
		service.Name,
		service.Description,
		service.Host,
		service.Port,
	)
	if err != nil {
		return nil, fmt.Errorf("error inserting new service: %s", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	service.Id = id
	return &service, nil

}

func (r *SQLiteRepository) GetAllServices() ([]backend.Service, error) {
	rows, err := r.db.Query("SELECT * from services")
	if err != nil {
		return nil, fmt.Errorf("error getting all services: %s", err)
	}
	defer rows.Close()

	var all []backend.Service
	for rows.Next() {
		var service backend.Service
		if err := rows.Scan(
			&service.Id,
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

func (r *SQLiteRepository) UpdateService(id int64, updated backend.Service) (*backend.Service, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid updated ID")
	}
	res, err := r.db.Exec(
		"INSERT INTO services(name, description, host, port)",
		updated.Name,
		updated.Description,
		updated.Host,
		updated.Port,
	)
	if err != nil {
		return nil, fmt.Errorf("error updating service with id of %d: %s", id, err)
	}

	rowsAffected, err := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("update failed: %s", err)
	}

	return &updated, nil

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
		return fmt.Errorf("failed to delete service")
	}
	return err
}

func OpenDatabaseFile(filename string) *sql.DB {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatalf("Error opening sqlite db: %s", err)
	}
	log.Printf("Opened database file")
	return db
}
