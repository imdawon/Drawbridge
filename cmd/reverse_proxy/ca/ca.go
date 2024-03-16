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
	"dhens/drawbridge/cmd/utils"
	"encoding/pem"
	"log"
	"log/slog"
	"math/big"
	"time"
)

type CA struct {
	CertPool             *x509.CertPool
	ClientTLSConfig      *tls.Config
	ServerTLSConfig      *tls.Config
	CertificateAuthority *x509.Certificate
	PrivateKey           crypto.PrivateKey
}

var CertificateAuthority *CA

func (c *CA) SetupCertificates() (err error) {
	caCertContents := utils.ReadFile("ca/ca.crt")
	caPrivKeyContents := utils.ReadFile("ca/ca.key")
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	serverKeyExists := utils.FileExists("ca/server-key.key")

	// Avoid generating new certificates and keys because we already have. Return TLS configs with the existing files.
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
			MinVersion:   tls.VersionTLS13,
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
	listeningAddressBytes := utils.ReadFile("config/listening_address.txt")
	listeningAddress := string(*listeningAddressBytes)
	// CA Cert, Server Cert, and Server key do not exist yet. We will generate them now, and save them to disk for reuse.
	// 1. Set up our CA certificate
	ca := x509.Certificate{
		DNSNames:     []string{listeningAddress, "localhost"},
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		// IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.ParseIP(listeningAddress)},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// If we want to listen on all interfaces, we need to add all interface IPs
	// to the list of acceptable IPs for the Drawbridge CA.
	if listeningAddress == "0.0.0.0" {
		ips, err := utils.GetDeviceIPs()
		if err != nil {
			return err
		}
		ca.IPAddresses = append(ca.IPAddresses, ips...)
		slog.Debug("IP ADDRESSES for CA: ", ca.IPAddresses)
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
		DNSNames: []string{listeningAddress, "localhost"},
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		// IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.ParseIP(listeningAddress)},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// If we want to listen on all interfaces, we need to add all interface IPs
	// to the list of acceptable IPs for the Drawbridge CA.
	if listeningAddress == "0.0.0.0" {
		ips, err := utils.GetDeviceIPs()
		if err != nil {
			return err
		}
		cert.IPAddresses = append(ca.IPAddresses, ips...)
		slog.Debug("IP ADDRESSES for server cert: ", ca.IPAddresses)
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
