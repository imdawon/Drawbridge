package emissary

import (
	"time"
)

// A device that can be allowed to access resources beyond Drawbridge.
type EmissaryClient struct {
	// TODO
	// make a uuid
	ID                             string
	Hostname                       string
	OperatingSystemVersion         string
	LastSuccessfulPolicyEvaluation time.Time
}
