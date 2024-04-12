package emissary

type Event struct {
	ID             string
	DeviceID       string
	ConnectionIP   string
	Type           string
	TargetService  string
	ConnectionType string
	Timestamp      string
}
