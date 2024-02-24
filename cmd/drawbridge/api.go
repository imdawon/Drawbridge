package drawbridge

import (
	"dhens/drawbridge/cmd/drawbridge/client"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
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
	AuthorizationPolicy client.AuthorizationPolicy
}

// When a request comes to our Emissary client api, this function verifies that the body matches the
// Drawbridge Authorization Policy.
// If authorized by passing the policy requirements, we will grant the Emissary client
// an mTLS key to be used by the Emissary client to access an http resource.
// If unauthorized, we send the Emissary client a 401.
func handleClientAuthorizationRequest(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("error reading client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientAuth := client.AuthorizationRequest{}
	err = json.Unmarshal(body, &clientAuth)
	if err != nil {
		log.Fatalf("error unmarshalling client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientIsAuthorized := client.TestAuthorizationPolicy.ClientIsAuthorized(clientAuth)
	if clientIsAuthorized {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "client auth success!")
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "client auth failure (unauthorized)!")
	}
}

// Set up an mTLS protected API to serve Emissary client requests.
// The Emissary API is mainly to handle authentication of Emissary clients,
// as well as provisioning mTLS certificates for them.
// Proxying requests for TCP and UDP traffic is handled by the reverse proxy.
func SetUpEmissaryAPI(hostAndPort string, ca *certificates.CA) {
	r := http.NewServeMux()
	r.HandleFunc("/emissary/v1/auth", handleClientAuthorizationRequest)
	server := http.Server{
		TLSConfig: ca.ServerTLSConfig,
		Addr:      hostAndPort,
		Handler:   r,
	}
	slog.Info(fmt.Sprintf("Starting Drawbridge api service on %s", server.Addr))

	// We pass "" into listen and serve since we have already configured cert and keyfile for server.
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func SetUpReverseProxy(ca *certificates.CA) {
	r := http.NewServeMux()
	r.HandleFunc("/", myHandler)
	server := http.Server{
		TLSConfig: ca.ServerTLSConfig,
		Addr:      "localhost:4443",
		Handler:   r,
	}
	slog.Info(fmt.Sprintf("Starting Drawbridge mTLS http server on %s", server.Addr))

	go func() {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}()

	ca.MakeClientHttpRequest(fmt.Sprintf("https://%s", server.Addr))
	ca.MakeClientAuthorizationRequest()
}

func myHandler(w http.ResponseWriter, req *http.Request) {
	slog.Debug(fmt.Sprintf("New request from %s", req.RemoteAddr))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success!")
}
