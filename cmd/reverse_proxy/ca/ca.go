package certificates

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"dhens/drawbridge/cmd/drawbridge/persistence"
	"dhens/drawbridge/cmd/utils"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"sync"
	"time"
)

type CA struct {
	CertPool                  *x509.CertPool
	ClientTLSConfig           *tls.Config
	ServerTLSConfig           *tls.Config
	CertificateAuthority      *x509.Certificate
	PrivateKey                crypto.PrivateKey
	DB                        *persistence.SQLiteRepository
	CertificateRevocationList map[string]uint8
}

var CertificateAuthority *CA

func (c *CA) SetupCertificates() error {
	caCertContents := utils.ReadFile("ca/ca.crt")
	caPrivKeyContents := utils.ReadFile("ca/ca.key")
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	serverKeyExists := utils.FileExists("ca/server-key.key")

	caCertPath := utils.CreateDrawbridgeFilePath("./ca/ca.crt")
	caKeyPath := utils.CreateDrawbridgeFilePath("./ca/ca.key")
	serverCertPath := utils.CreateDrawbridgeFilePath("./ca/ca.crt")
	serverKeyPath := utils.CreateDrawbridgeFilePath("./ca/ca.key")

	// Avoid generating new certificates and keys because we already have. Return TLS configs with the existing files.
	if caCertContents != nil && serverCertExists && serverKeyExists && caPrivKeyContents != nil {
		slog.Info("TLS Certs & Keys already exist. Loading them from disk...")
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM(*caCertContents)

		// Combine the keypair for the CA certificate
		caCert, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
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

		// Load revoked certs into CRL from db.
		revokedCerts, err := c.DB.GetAllEmissaryClientCertificates()
		if err != nil {
			return fmt.Errorf("errpr getting all emissary client certs: %w", err)
		}
		c.CertificateRevocationList = revokedCerts
		// Read the key pair to create certificate
		serverCert, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath)
		if err != nil {
			log.Fatal("Error loading server cert and key files: ", err)
		}

		c.ServerTLSConfig = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    certpool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS13,
			// Ensure device cert is valid during handshake
			VerifyPeerCertificate: c.verifyEmissaryCertificate,
		}
		c.ClientTLSConfig = &tls.Config{
			RootCAs:      certpool,
			Certificates: []tls.Certificate{serverCert},
			MinVersion:   tls.VersionTLS13,
		}

		// Terminate function early as we have all of the cert and key data we need.
		slog.Info("Loaded TLS Certs & Keys")
		return nil
	}

	// Read the listening address set by the Drawbridge admin. This is important as it sets the DNSNames fields
	// for the certificate authority and server certificates, which is necessary to ensure the Emissary clients
	// can validate the Drawbridge server they are connecting to, is, in fact, the correct one.
	listeningAddress, err := c.DB.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		slog.Error("Database", slog.Any("Error: %s", err))
	}
	isLAN := drawbridgeListeningAddressIsLAN(net.ParseIP(*listeningAddress))
	slog.Debug("Drawbridge listening address type", slog.Bool("isLAN", isLAN))
	// CA Cert, Server Cert, and Server key do not exist yet. We will generate them now, and save them to disk for reuse.
	// 1. Set up our CA certificate
	ca := x509.Certificate{
		DNSNames:     []string{*listeningAddress, "localhost"},
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.ParseIP(*listeningAddress)},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Listen on all interfaces if the listening address isn't an IANA private IPv4 address e.g if the user
	// uses their WAN IP address.
	// Otherwise we only listen on the LAN address and local loopback network as the user wants.
	if !isLAN {
		ips, err := utils.GetDeviceIPs()
		if err != nil {
			return err
		}
		ca.IPAddresses = append(ca.IPAddresses, ips...)
		slog.Debug("Certificates and Keys", slog.String("CA Allowed IP Addresses", fmt.Sprintf("%s", ca.IPAddresses)))

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
		DNSNames: []string{*listeningAddress, "localhost"},
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.ParseIP(*listeningAddress)},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Listen on all interfaces if the listening address isn't an IANA private IPv4 address e.g if the user
	// uses their WAN IP address.
	// Otherwise we only listen on the LAN address and local loopback network as the user wants.
	if !isLAN {
		ips, err := utils.GetDeviceIPs()
		if err != nil {
			return err
		}
		cert.IPAddresses = append(ca.IPAddresses, ips...)
		slog.Debug("Certificates and Keys", slog.String("Server Certificate Allowed IP Addresses", fmt.Sprintf("%s", ca.IPAddresses)))
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
		// Ensure device cert is valid during handshake
		VerifyPeerCertificate: c.verifyEmissaryCertificate,
	}

	c.ClientTLSConfig = &tls.Config{
		RootCAs:      certpool,
		Certificates: []tls.Certificate{serverCert},
	}

	c.CertificateRevocationList = make(map[string]uint8)

	return nil
}

// THIS FUNCTION NEEDS TO BE FAST TO NOT DELAY HANDSHAKE
// Run for every Drawbridge + Emissary handshake to verify the presented cert is not revoked.
func (c *CA) verifyEmissaryCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// Parse the peer certificate
	// PEM encode
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rawCerts[0],
	})

	// Calculate the SHA-256 hash of the peer certificate
	hash := sha256.Sum256(caPEM.Bytes())
	hexHash := hex.EncodeToString(hash[:])

	// Check if the certificate hash is in the revocation list
	if c.CertificateRevocationList[hexHash] == 1 {
		slog.Debug("peer cert is REVOKED")
		return errors.New("peer certificate is revoked")
	}

	// Additional certificate verification checks can be added here
	slog.Debug("peer cert is VALID")

	return nil
}

var revokedCertsMutex sync.RWMutex

// RevokeCertInCertificateRevocationList adds a certificate to the revoked certificates list
func (c *CA) RevokeCertInCertificateRevocationList(shaCert string) {
	revokedCertsMutex.Lock()
	defer revokedCertsMutex.Unlock()
	c.CertificateRevocationList[shaCert] = 1
}

// RevokeCertInCertificateRevocationList adds a certificate to the revoked certificates list
func (c *CA) UnRevokeCertInCertificateRevocationList(shaCert string) {
	revokedCertsMutex.Lock()
	defer revokedCertsMutex.Unlock()
	delete(c.CertificateRevocationList, shaCert)
}

// If someone is listening on a LAN address, we don't want to listen on all interfaces like we do if someone uses their
// public WAN address, for example.
func drawbridgeListeningAddressIsLAN(listeningAddress net.IP) bool {
	_, ten, _ := net.ParseCIDR("10.0.0.0/8")
	_, oneNineTwo, _ := net.ParseCIDR("192.168.0.0/16")
	_, oneSevenTwo, _ := net.ParseCIDR("192.168.1.0/12")
	return ten.Contains(listeningAddress) || oneNineTwo.Contains(listeningAddress) || oneSevenTwo.Contains(listeningAddress)
}
