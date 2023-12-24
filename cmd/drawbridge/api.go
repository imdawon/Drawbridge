package drawbridge

import (
	auth "dhens/drawbridge/cmd/drawbridge/client"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// A service that Drawbridge will protect by only allowing access from authorized machines running the Emissary client.
// In the future, a Client Policy can be assigned to a Protected Service, allowing for different requirements for different Protected Services.
type ProtectedService struct {
	ID                  int64
	Name                string `schema:"service-name" json:"service-name"`
	Description         string `schema:"service-description" json:"service-description"`
	Host                string `schema:"service-host" json:"service-host"`
	Port                uint16 `schema:"service-port" json:"service-port"`
	ClientPolicyID      int64  `schema:"service-policy-id,omitempty" json:"service-policy-id,omitempty"`
	AuthorizationPolicy auth.AuthorizationPolicy
}

func handleClientAuthorizationRequest(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Fatalf("error reading client auth request: %s", err)
		c.JSON(500, gin.H{
			"success": false,
		})
	}

	clientAuth := &auth.AuthorizationRequest{}
	err = json.Unmarshal(body, &clientAuth)
	if err != nil {
		log.Fatalf("error unmarshalling client auth request: %s", err)
		c.JSON(500, gin.H{
			"success": false,
		})
	}

	auth.TestAuthorizationPolicy.ClientIsAuthorized(*clientAuth)

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
