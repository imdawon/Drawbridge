package services

type RunningProtectedService struct {
	Service ProtectedService
}

// A service that Drawbridge will protect by only allowing access from authorized machines running the Emissary client.
// In the future, a Client Policy can be assigned to a Protected Service, allowing for different requirements for different Protected Services.
type ProtectedService struct {
	ID             int64
	Name           string `schema:"service-name" json:"service-name"`
	Description    string `schema:"service-description" json:"service-description"`
	Host           string `schema:"service-host" json:"service-host"`
	Port           uint16 `schema:"service-port" json:"service-port"`
	ClientPolicyID int64  `schema:"service-policy-id,omitempty" json:"service-policy-id,omitempty"`
	// AuthorizationPolicy  authorization.Policy `schema:"authorization-policy,omitempty" json:"authorization-policy,omitempty"`
}
