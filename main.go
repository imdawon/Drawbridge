package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	CACertFilePath = "./contoso.crt"
	CertFilePath   = "./server-crt.pem"
	KeyFilePath    = "./server-key.pem"
)

func httpRequestHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello,World!\n"))
}
func main() {
	// load tls certificates
	serverTLSCert, err := tls.LoadX509KeyPair(CertFilePath, KeyFilePath)
	if err != nil {
		log.Fatalf("Error loading certificate and key file: %v", err)
	}
	// Configure the server to trust TLS client cert issued by your CA.
	certPool := x509.NewCertPool()
	if caCertPEM, err := os.ReadFile(CACertFilePath); err != nil {
		panic(err)
	} else if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
		panic("invalid cert in CA PEM")
	}
	tlsConfig := &tls.Config{
		ServerName:   "server.localhost",
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		RootCAs:      certPool,
		Certificates: []tls.Certificate{serverTLSCert},
	}
	server := http.Server{
		Addr:      "localhost:443",
		Handler:   http.HandlerFunc(httpRequestHandler),
		TLSConfig: tlsConfig,
	}
	defer server.Close()
	fmt.Printf("Server starting...")
	log.Fatal(server.ListenAndServeTLS("", ""))
}
