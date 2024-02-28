package certificates

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"dhens/drawbridge/cmd/drawbridge/client"
	"dhens/drawbridge/cmd/utils"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"
)

type CA struct {
	CertPool             *x509.CertPool
	ClientTLSConfig      *tls.Config
	ServerTLSConfig      *tls.Config
	CertificateAuthority *x509.Certificate
	PrivateKey           crypto.PrivateKey
}

func (c *CA) SetupCertificates() (err error) {
	caCertContents := utils.ReadFile("ca/ca.crt")
	caPrivKeyContents := utils.ReadFile("ca/ca.key")
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	serverKeyExists := utils.FileExists("ca/server-key.key")

	// Avoid generating new certificates and keys. Return TLS configs with the existing files.
	if caCertContents != nil && serverCertExists && serverKeyExists && caPrivKeyContents != nil {
		slog.Info("TLS Certs & Keys already exist. Loading them from disk...")
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM(*caCertContents)

		// Combine the keypair for the CA certificate
		caCert, err := tls.LoadX509KeyPair("ca/ca.crt", "ca/ca.key")
		if err != nil {
			log.Fatal("Error loading CA cert and key files: ", err)
		}
		c.PrivateKey = caCert.PrivateKey

		// We have to decode the certificate into the ASN.1 DER format before attempting to parse it.
		// Otherwise, we will error out when parsing.
		block, _ := pem.Decode(*caCertContents)
		if block == nil || block.Type != "CERTIFICATE" {
			log.Fatal("failed to decode PEM block containing CERTIFICATE")
		}
		c.CertificateAuthority, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatal("Error parsing CA cert: ", err)
		}
		// Read the key pair to create certificate
		serverCert, err := tls.LoadX509KeyPair("ca/server-cert.crt", "ca/server-key.key")
		if err != nil {
			log.Fatal("Error loading server cert and key files: ", err)
		}

		c.ServerTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    certpool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		}

		c.ClientTLSConfig = &tls.Config{
			RootCAs:      certpool,
			Certificates: []tls.Certificate{serverCert},
		}

		// Terminate function early as we have all of the cert and key data we need.
		slog.Info("Loaded TLS Certs & Keys")
		return nil
	}

	// CA Cert, Server Cert, and Server key do not exist yet. We will generate them now, and save them to disk for reuse.
	// 1. Set up our CA certificate
	ca := x509.Certificate{
		DNSNames:     []string{"localhost"},
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

	c.CertificateAuthority = &ca

	// Create our private and public key for the Certificate Authority.
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}

	// Create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, &ca, &ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// PEM encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	err = utils.SaveFile("ca.crt", caPEM.String(), "ca")
	if err != nil {
		return err
	}
	// Save to Emissary certs and key folder so we don't have to do it on-demand when a Drawbridge admin generates a cert and key.
	err = utils.SaveFile("ca.crt", caPEM.String(), "./emissary_certs_and_key_here")
	if err != nil {
		return err
	}

	caPrivKeyPEMBytes, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: caPrivKeyPEMBytes,
	})
	err = utils.SaveFile("ca.key", caPrivKeyPEM.String(), "ca")
	if err != nil {
		return err
	}

	serverCert, err := tls.X509KeyPair(caPEM.Bytes(), caPrivKeyPEM.Bytes())
	if err != nil {
		return err
	}

	// 2. Set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		// TODO: Must be domain name or IP during user dash setup
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
		return err
	}

	// Create the server certificate and sign it with our CA.
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, &ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	err = utils.SaveFile("server-cert.crt", certPEM.String(), "ca")
	if err != nil {
		return err
	}

	certPrivKeyPEMBytes, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		return err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certPrivKeyPEMBytes,
	})

	err = utils.SaveFile("server-key.key", certPrivKeyPEM.String(), "ca")
	if err != nil {
		return err
	}

	// Store the CA private key in our CA struct for use later during runtime.
	c.PrivateKey = caPrivKey

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())

	c.ServerTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    certpool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	c.ClientTLSConfig = &tls.Config{
		RootCAs:      certpool,
		Certificates: []tls.Certificate{serverCert},
	}

	return nil
}

func (c *CA) MakeClientHttpRequest(url string) {
	// Communicate with the http server using an http.Client configured to trust our CA.
	transport := &http.Transport{
		TLSClientConfig: c.ClientTLSConfig,
	}
	http := http.Client{
		Transport: transport,
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("GET request to %s failed: %s", url, err)
	}
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body response from reverse proxy request: %s", err)
	}
	body := strings.TrimSpace(string(respBodyBytes[:]))
	slog.Info(fmt.Sprintf("client request body: %s", body))
}

func (c *CA) MakeClientAuthorizationRequest() {
	// Communicate with the http server using an http.Client configured to trust our Drawbridge CA.
	transport := &http.Transport{
		TLSClientConfig: c.ClientTLSConfig,
	}
	http := http.Client{
		Transport: transport,
	}

	authorizationRequest := client.TestAuthorizationRequest
	out, err := json.Marshal(authorizationRequest)
	if err != nil {
		log.Fatalf("failed to marshal auth request: %s", err)
	}

	resp, err := http.Post("https://localhost:3001/emissary/v1/auth", "application/json", bytes.NewBuffer(out))
	if err != nil {
		log.Fatalf("POST to auth endpoint failed: %s", err)
	}
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading body response from client auth request: %s", err)
	}
	body := strings.TrimSpace(string(respBodyBytes[:]))
	slog.Debug(fmt.Sprintf("client request body: %s", body))
}

// An Emissary TCP Mutual TLS Key is used to allow the Emissary Client to connect to Drawbridge directly.
// The user will connect to the local proxy server the Emissary Client creates and all traffic will then flow
// through Drawbridge.
func (c *CA) CreateEmissaryClientTCPMutualTLSKey(clientId string) error {
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	if !serverCertExists {
		slog.Error("Unable to create new Emissary Client TCP mTLS key. Server certificate does not exist!")
	}

	clientCert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		// TODO: Must be domain name or IP during user dash setup
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
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	clientCertPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}

	// Create the client certificate and sign it with our CA private key.
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientCert, c.CertificateAuthority, &clientCertPrivKey.PublicKey, c.PrivateKey)
	if err != nil {
		slog.Error(fmt.Sprintf("%s", err))
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientCertBytes,
	})
	// Save the file to disk for use by an Emissary client. This should be later used and saved in the db for downloading later.
	err = utils.SaveFile("emissary-mtls-tcp.crt", certPEM.String(), "./emissary_certs_and_key_here")
	if err != nil {
		return err
	}

	certPrivKeyPEMBytes, err := x509.MarshalECPrivateKey(clientCertPrivKey)
	if err != nil {
		return err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certPrivKeyPEMBytes,
	})
	// Save the file to disk for use by an Emissary client. This should be later used and saved in the db for downloading later.
	err = utils.SaveFile("emissary-mtls-tcp.key", certPrivKeyPEM.String(), "./emissary_certs_and_key_here")
	if err != nil {
		slog.Error(fmt.Sprintf("Error saving x509 keypair for Emissary client to disk: %s", err))
	}

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(certPEM.Bytes())
	c.ClientTLSConfig.Certificates = append(c.ClientTLSConfig.Certificates, serverCert)

	return nil
}
