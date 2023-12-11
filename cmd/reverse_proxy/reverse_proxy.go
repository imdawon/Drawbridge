package proxy

import (
	proxy "dhens/drawbridge/cmd/reverse_proxy/ca"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func SetUpReverseProxy() {
	ca := &proxy.CA{}
	err := ca.SetupRootCA()
	if err != nil {
		log.Fatalf("Error setting up root CA: %s", err)
	}

	r := mux.NewRouter()
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
