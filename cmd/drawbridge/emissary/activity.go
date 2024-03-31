package emissary

import (
	"database/sql"
	"time"
)

type Event struct {
	ID             int64
	DeviceID       string
	DeviceIP       string
	Type           string
	TargetService  sql.NullString
	ConnectionType sql.NullString
	Timestamp      time.Time
}
