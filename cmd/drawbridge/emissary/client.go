package emissary

// A device that can be allowed to access resources beyond Drawbridge.
type EmissaryClient struct {
	ID   string
	Name string
	// The mTLS certificate Emissary uses to connect to Drawbridge.
	DrawbridgeCertificate string
	Revoked               uint8
}

type DeviceCertificate struct {
	Revoked  uint8
	DeviceID string
}
