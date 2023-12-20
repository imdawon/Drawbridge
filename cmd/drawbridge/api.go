package drawbridge

import (
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

// A service that Drawbridge will protect by only allowing access from authorized machines running the Emissary client.
// In the future, a Client Policy can be assigned to a Protected Service, allowing for different requirements for different Protected Services.
type ProtectedService struct {
	ID             int64
	Name           string `schema:"service-name" json:"service-name"`
	Description    string `schema:"service-description" json:"service-description"`
	Host           string `schema:"service-host" json:"service-host"`
	Port           uint16 `schema:"service-port" json:"service-port"`
	ClientPolicyID int64  `schema:"service-policy-id,omitempty" json:"service-policy-id,omitempty"`
}

// The policy that Drawbridge will use to evaluate if it will allow access to an Emissary client.
type AuthorizationPolicy struct {
	ID           int64
	Name         string        `schema:"policy-name" json:"policy-name"`
	Description  string        `schema:"policy-description" json:"policy-description"`
	Requirements Authorization `schema:"policy-requirements" json:"policy-requirements"`
}

// An Authorization is all of the characteristics collected about a machine running the Emissary client.
// If an Authorization passes the Requirements of an Authorization Policy, Drawbridge allow it to access protected resources.
type Authorization struct {
	WANIP        net.IP `json:"wan-ip"`
	OSType       string `json:"os-type"`
	SerialNumber string `json:"serial-number"`
}

var TestAuthorizationPolicy = AuthorizationPolicy{
	Name:        "Allow Personal Machine",
	Description: "",
	Requirements: Authorization{
		WANIP:        net.IPv4(8, 8, 8, 8),
		OSType:       "Windows",
		SerialNumber: "00000-00000-00000-00000",
	},
}

func (arv *AuthorizationPolicy) clientIsAuthorized(clientAuthorization Authorization) bool {
	authorizationPolicyRequirements := reflect.ValueOf(arv.Requirements)
	for i := 0; i < authorizationPolicyRequirements.NumField(); i++ {
		fmt.Printf("value: %v", authorizationPolicyRequirements.Field(i))

	}
	return true
}

func handleClientAuthorizationRequest(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Fatalf("error reading client auth request: %s", err)
		c.JSON(500, gin.H{
			"success": false,
		})
	}

	clientAuth := &Authorization{}
	err = json.Unmarshal(body, &clientAuth)
	if err != nil {
		log.Fatalf("error unmarshalling client auth request: %s", err)
		c.JSON(500, gin.H{
			"success": false,
		})
	}

	TestAuthorizationPolicy.clientIsAuthorized(*clientAuth)

	c.JSON(200, gin.H{
		"message": "pong",
		"success": true,
	})
}

func SetUp(hostAndPort string) {
	log.Printf("Starting drawbridge api service on %s", hostAndPort)

	r := gin.Default()
	r.POST("/emissary/v1/auth", handleClientAuthorizationRequest)

	r.Run(hostAndPort)
}

func SetUpReverseProxy(ca *certificates.CA) {
	r := http.NewServeMux()
	r.HandleFunc("/", myHandler)
	server := http.Server{
		TLSConfig: ca.ServerTLSConfig,
		Addr:      "localhost:4443",
		Handler:   r,
	}
	log.Printf("Listening Drawbridge reverse rpoxy at %s", server.Addr)

	go func() {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}()

	ca.MakeClientRequest(fmt.Sprintf("https://%s", server.Addr))
}

func myHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("New request from %s", req.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success!")
}
