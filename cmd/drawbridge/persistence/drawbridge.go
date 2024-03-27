package persistence

import (
	"fmt"
	"strings"
)

func (r *SQLiteRepository) MigrateDrawbridgeConfig() error {
	query := `
	CREATE TABLE IF NOT EXISTS drawbridge_config(
		setting TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL
	);
	`

	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteRepository) CreateNewDrawbridgeConfigSettings(setting, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO drawbridge_config(setting, value) values(?,?)",
		setting,
		value,
	)
	if err != nil {
		return fmt.Errorf("error creating new emissary client: %s", err)
	}

	return nil

}

func (r *SQLiteRepository) GetDrawbridgeConfigValueByName(setting string) (*string, error) {
	rows, err := r.db.Query("SELECT * FROM drawbridge_config WHERE setting = ?", setting)
	if err != nil {
		if strings.Contains(err.Error(), "no such column") {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting drawbridge %s setting: %s", setting, err)
	}
	defer rows.Close()

	var settingValue string
	var value string
	for rows.Next() {
		if err := rows.Scan(
			&settingValue,
			&value,
		); err != nil {
			return nil, fmt.Errorf("error scanning drawbridge setting database row into string: %s", err)
		}
	}
	return &value, nil
}

func (r *SQLiteRepository) UpdateDrawbridgeConfigSettingByName(setting string, value string) error {
	if setting == "" || value == "" {
		return fmt.Errorf("name ('%s') and/or value ('%s') cannot be 0 character strings", setting, value)
	}
	res, err := r.db.Exec(
		"UPDATE drawbridge_config SET value = ? WHERE name = ?",
		setting,
		value,
	)
	if err != nil {
		return fmt.Errorf("error updating drawbridge '%s' setting: %s", setting, err)
	}

	rowsAffected, err := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no drawbridge setting rows updated: %s", err)
	}

	return nil

}

func (r *SQLiteRepository) DeleteDrawbridgeConfigSetting(setting string) error {
	res, err := r.db.Exec("DELETE FROM drawbridge_config WHERE name = ?", setting)
	if err != nil {
		return fmt.Errorf("error deleting drawbridge %s setting: %s", setting, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("failed to delete drawbridge setting")
	}
	return err
}
