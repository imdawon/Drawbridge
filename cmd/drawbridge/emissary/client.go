package emissary

import (
	"time"

	"github.com/google/uuid"
)

// A device that can be allowed to access resources beyond Drawbridge.
type EmissaryClient struct {
	ID                             uuid.UUID
	Hostname                       string
	OperatingSystemVersion         string
	LastSuccessfulPolicyEvaluation time.Time
}
