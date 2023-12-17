package drawbridge

import (
	"log"
	"net"

	"github.com/gin-gonic/gin"
)

// A service that Drawbridge will protect by only allowing access from authorized machines running the Emissary client.
// In the future, a Client Policy can be assigned to a Protected Service, allowing for different requirements for different Protected Services.
type ProtectedService struct {
	ID             int64
	Name           string `schema:"service-name"`
	Description    string `schema:"service-description"`
	Host           string `schema:"service-host"`
	Port           uint16 `schema:"service-port"`
	ClientPolicyID int64  `schema:"service-policy-id,omitempty"`
}

// The policy that Drawbridge will use to evaluate if it will allow access to an Emissary client.
type ClientPolicy struct {
	ID                       int64
	Name                     string                   `schema:"policy-name"`
	Description              string                   `schema:"policy-description"`
	AuthorizationRequirement AuthorizationRequirement `schema:"policy-requirements"`
}

// The values required by a Client Policy to authorize an Emissary client and allow it to access to resources protected by Drawbridge.
// An Emissary client will upload a json blob containing the following information,
// and will be checked against the Client Policy assigned to the resource requested by Emissary.
type AuthorizationRequirement struct {
	IP           net.IP `json:"wan-ip"`
	OSType       string `json:"os-type"`
	SerialNumber string `json:"serial-number"`
}

func validateClientAuthRequest(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func SetUp(hostAndPort string) {
	log.Printf("Starting drawbridge api service on %s", hostAndPort)

	r := gin.Default()
	r.POST("/emissary/v1/auth", validateClientAuthRequest)

	r.Run(hostAndPort)
}

// func Create
