package emissary

// A device that can be allowed to access resources beyond Drawbridge.
type EmissaryClient struct {
	ID      string
	Name    string
	Revoked uint8
}
