package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"dhens/drawbridge/cmd/utils"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var (
	CACertPool      *x509.CertPool
	ClientTLSConfig *tls.Config
	serverTLSConfig *tls.Config
)

func main() {
	var err error
	serverTLSConfig, ClientTLSConfig, err = setupRootCA()
	if err != nil {
		log.Fatalf("Error setting up root CA: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/", myHandler)
	server := http.Server{
		TLSConfig: serverTLSConfig,
		Addr:      "localhost:4443",
		Handler:   r,
	}
	log.Printf("Listening Drawbridge reverse rpoxy at %s", server.Addr)

	go func() {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}()

	makeClientRequest(fmt.Sprintf("https://%s", server.Addr))
}

func myHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("New request from %s", req.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success!")
}

func setupRootCA() (serverTLSConfig *tls.Config, clientTLSConfig *tls.Config, err error) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	err = utils.SaveFile("ca.pem", caPEM.String(), "./cmd/reverse_proxy/ca")
	if err != nil {
		return nil, nil, err
	}
	caPrivKeyPEMBytes, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: caPrivKeyPEMBytes,
	})

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		// Must be domain name or IP during user dash setup
		DNSNames: []string{"localhost"},
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Create the server certificate and sign it with our CA.
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	err = utils.SaveFile("server-cert.pem", certPEM.String(), "./cmd/reverse_proxy/ca")
	if err != nil {
		return nil, nil, err
	}

	certPrivKeyPEMBytes, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certPrivKeyPEMBytes,
	})

	err = utils.SaveFile("server-key.pem", certPrivKeyPEM.String(), "./cmd/reverse_proxy/ca")
	if err != nil {
		return nil, nil, err
	}
	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, nil, err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())

	serverTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    certpool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	clientTLSConfig = &tls.Config{
		RootCAs:      certpool,
		Certificates: []tls.Certificate{serverCert},
	}

	return
}

func makeClientRequest(url string) {
	// communicate with the server using an http.Client configured to trust our CA
	transport := &http.Transport{
		TLSClientConfig: ClientTLSConfig,
	}
	http := http.Client{
		Transport: transport,
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Cannot GET reverse proxy endpoint: %s", err)
	}
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body response from reverse proxy request: %s", err)
	}
	body := strings.TrimSpace(string(respBodyBytes[:]))
	fmt.Printf("client request body: %s\n", body)
}
