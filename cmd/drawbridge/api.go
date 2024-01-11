package drawbridge

import (
	auth "dhens/drawbridge/cmd/drawbridge/client"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	AuthorizationPolicy auth.AuthorizationPolicy
}

func handleClientAuthorizationRequest(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("error reading client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientAuth := auth.AuthorizationRequest{}
	err = json.Unmarshal(body, &clientAuth)
	if err != nil {
		log.Fatalf("error unmarshalling client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientIsAuthorized := auth.TestAuthorizationPolicy.ClientIsAuthorized(clientAuth)
	if clientIsAuthorized {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "success!")

	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "unauthorized!")

	}
}

func SetUp(hostAndPort string, ca *certificates.CA) {
	r := http.NewServeMux()
	r.HandleFunc("/emissary/v1/auth", handleClientAuthorizationRequest)
	server := http.Server{
		TLSConfig: ca.ServerTLSConfig,
		Addr:      hostAndPort,
		Handler:   r,
	}
	log.Printf("Starting Drawbridge api service on %s", server.Addr)

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
	log.Printf("Listening Drawbridge reverse rpoxy at %s", server.Addr)

	go func() {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}()

	ca.MakeClientRequest(fmt.Sprintf("https://%s", server.Addr))
	ca.MakeClientAuthorizationRequest()
}

func myHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("New request from %s", req.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success!")
}
