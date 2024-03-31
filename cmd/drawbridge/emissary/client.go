package emissary

import (
	"time"
)

// A device that can be allowed to access resources beyond Drawbridge.
type EmissaryClient struct {
	// TODO
	// make a uuid
	ID                             string
	Name                           string
	OperatingSystemVersion         string
	LastSuccessfulPolicyEvaluation time.Time
}
