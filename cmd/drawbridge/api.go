package drawbridge

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"dhens/drawbridge/cmd/drawbridge/emissary/authorization"
	flagger "dhens/drawbridge/cmd/flags"
	certificates "dhens/drawbridge/cmd/reverse_proxy/ca"
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

type Settings struct {
	ListenerAddress string `schema:"listener-address"`
}

// Used by the frontend controller to execute Drawbridge functions.
// ProtectedServices contains a map of listeners running for each Protected Service.
// The int key is the ID of the service as stored in the database.
type Drawbridge struct {
	CA                *certificates.CA
	ProtectedServices map[int64]RunningProtectedService
}

type RunningProtectedService struct {
	Service ProtectedService
}

// A service that Drawbridge will protect by only allowing access from authorized machines running the Emissary client.
// In the future, a Client Policy can be assigned to a Protected Service, allowing for different requirements for different Protected Services.
type ProtectedService struct {
	ID                  int64
	Name                string               `schema:"service-name" json:"service-name"`
	Description         string               `schema:"service-description" json:"service-description"`
	Host                string               `schema:"service-host" json:"service-host"`
	Port                uint16               `schema:"service-port" json:"service-port"`
	ClientPolicyID      int64                `schema:"service-policy-id,omitempty" json:"service-policy-id,omitempty"`
	AuthorizationPolicy authorization.Policy `schema:"authorization-policy,omitempty" json:"authorization-policy,omitempty"`
}

// When a request comes to our Emissary client api, this function verifies that the body matches the
// Drawbridge Authorization Policy.
// If authorized by passing the policy requirements, we will grant the Emissary client
// an mTLS key to be used by the Emissary client to access an http resource.
// If unauthorized, we send the Emissary client a 401.
func (d *Drawbridge) handleClientAuthorizationRequest(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Fatalf("error reading client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientAuth := authorization.EmissaryRequest{}
	err = json.Unmarshal(body, &clientAuth)
	if err != nil {
		log.Fatalf("error unmarshalling client auth request: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "server error!")
	}

	clientIsAuthorized := authorization.TestPolicy.ClientIsAuthorized(clientAuth)
	if clientIsAuthorized {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "client auth success!")
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "client auth failure (unauthorized)!")
	}
}

// Set up an mTLS-protected API to serve Emissary client requests.
// The Emissary API is mainly to handle authentication of Emissary clients,
// as well as provisioning mTLS certificates for them.
// Proxying requests for TCP and UDP traffic is handled by the reverse proxy.
func (d *Drawbridge) SetUpEmissaryAPI(hostAndPort string) {
	r := http.NewServeMux()
	r.HandleFunc("/emissary/v1/auth", d.handleClientAuthorizationRequest)
	server := http.Server{
		TLSConfig: d.CA.ServerTLSConfig,
		Addr:      hostAndPort,
		Handler:   r,
	}
	slog.Info(fmt.Sprintf("Starting Emissary API server on http://%s", server.Addr))

	// We pass "" into listen and serve since we have already configured cert and keyfile for server.
	log.Fatalf("Error starting Emissary API server: %s", server.ListenAndServeTLS("", ""))
}

func (d *Drawbridge) SetUpCAAndDependentServices(protectedServices []ProtectedService) {
	certificates.CertificateAuthority = &certificates.CA{}
	err := certificates.CertificateAuthority.SetupCertificates()
	if err != nil {
		log.Fatalf("Error setting up root CA: %s", err)
	}
	// Set certificate authority for Drawbridge. We access the CA from Drawbridge from this point on.
	d.CA = certificates.CertificateAuthority

	// Start TCP and UDP listeners for each Drawbridge Protected Service.
	for _, service := range protectedServices {
		d.AddNewProtectedService(service)
	}

	go d.SetUpProtectedServiceTunnel()

	d.SetUpEmissaryAPI(flagger.FLAGS.BackendAPIHostAndPort)
}

// An Emissary TCP Mutual TLS Key is used to allow the Emissary Client to connect to Drawbridge directly.
// The user will connect to the local proxy server the Emissary Client creates and all traffic will then flow
// through Drawbridge.
func (d *Drawbridge) CreateEmissaryClientTCPMutualTLSKey(clientId string) error {
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	if !serverCertExists {
		slog.Error("Unable to create new Emissary Client TCP mTLS key. Server certificate does not exist!")
	}

	listeningAddressBytes := utils.ReadFile("config/listening_address.txt")
	listeningAddress := string(*listeningAddressBytes)

	clientCert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		// TODO: Must be domain name or IP during user dash setup
		Subject: pkix.Name{
			Organization:  []string{"Drawbridge"},
			Country:       []string{""},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    listeningAddress,
		},
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
	clientCertBytes, err := x509.CreateCertificate(
		rand.Reader,
		clientCert,
		d.CA.CertificateAuthority,
		&clientCertPrivKey.PublicKey,
		d.CA.PrivateKey,
	)
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

	emissaryCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(certPEM.Bytes())
	//  Add Emissary mTLS certificate to list of acceptable client certificates.
	d.CA.ClientTLSConfig.Certificates = append(d.CA.ClientTLSConfig.Certificates, emissaryCert)

	return nil
}

func (d *Drawbridge) AddNewProtectedService(protectedService ProtectedService) error {
	d.ProtectedServices[protectedService.ID] = RunningProtectedService{
		Service: protectedService,
	}
	return nil
}

func (d *Drawbridge) StopRunningProtectedService(id int64) {
	delete(d.ProtectedServices, id)
}

// This is the service the Emissary client connects to when it wants to access a Protected Service.
// It needs to take the Emissary connection and route it to the proper Protected Service.
func (d *Drawbridge) SetUpProtectedServiceTunnel() error {
	// The host and port this tcp server will listen on.
	// This is distinct from the ProtectedService "Host" field, which is the remote address of the actual service itself.
	addressAndPortBytes := utils.ReadFile("config/listening_address.txt")
	addressAndPort := fmt.Sprintf("%s:3100", string(*addressAndPortBytes))
	slog.Info(fmt.Sprintf("Starting Drawbridge reverse proxy tunnel. Emissary clients can reach Drawbridge at %s", addressAndPort))
	l, err := tls.Listen("tcp", "0.0.0.0:3100", d.CA.ServerTLSConfig)

	if err != nil {
		slog.Error(fmt.Sprintf("Reverse proxy TCP Listen failed: %s", err))
	}

	defer l.Close()

	for {
		// Wait and accept connections that present a valid mTLS certificate.
		conn, _ := l.Accept()

		// Handle new connection in a new go routine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(clientConn net.Conn) {
			// Read incoming data
			buf := make([]byte, 256)
			_, err := conn.Read(buf)
			if err != nil {
				fmt.Println(err)
				return
			}
			buf = bytes.Trim(buf, "\x00")
			// Print the incoming data - for debugging
			fmt.Printf("Received: %s\n", buf)

			var emissaryRequestValue string
			emissaryRequestPayload := string(buf[:])
			if strings.Contains(emissaryRequestPayload, "PS_CONN") {
				emissaryRequestValue = strings.TrimPrefix(emissaryRequestPayload, "PS_CONN")
				emissaryRequestValue = strings.TrimSpace(emissaryRequestValue)
				emissaryRequestValue = strings.TrimSpace(emissaryRequestValue)
				// May be used later after we standardize how and when to read the tcp connection into the buf above.
				// d.getRequestProtectedServiceName(clientConn)

				requestedServiceAddress := d.getProtectedServiceAddressByName(emissaryRequestValue)

				// Proxy traffic to the actual service the Emissary client is trying to connect to.
				var dialer net.Dialer
				resourceConn, err := dialer.Dial("tcp", requestedServiceAddress)
				// This can happen if the Drawbridge admin deletes a Protected Service while it is running.
				// The net.Listener will be closed and any remaining Accept operations are blocked and return errors.
				if err != nil {
					slog.Error("Failed to tcp dial to actual target service", err)
				}

				slog.Info(fmt.Sprintf("TCP Accept from Emissary client: %s\n", clientConn.RemoteAddr()))
				// Copy data back and from client and server.
				go io.Copy(resourceConn, clientConn)
				io.Copy(clientConn, resourceConn)
				// Shut down the connection.
				clientConn.Close()

			} else {
				// On a new connection, write available services to TCP connection so Emissary can know which
				// d.ProtectedServices
				var serviceList string
				for _, value := range d.ProtectedServices {
					serviceList += fmt.Sprintf("%s,", value.Service.Name)
				}
				serviceConnectCommand := fmt.Sprintf("PS_LIST: %s", serviceList)
				clientConn.Write([]byte(serviceConnectCommand))
			}
		}(conn)
	}
}

func (d *Drawbridge) getRequestProtectedServiceName(clientConn net.Conn) (string, error) {
	bytes, err := io.ReadAll(io.LimitReader(clientConn, 64))
	if err != nil {
		return "", err
	}

	return string(bytes[:]), nil
}

func (d *Drawbridge) getProtectedServiceAddressByName(protectedServiceName string) string {
	for _, service := range d.ProtectedServices {
		if service.Service.Name == protectedServiceName {
			protectedService := d.ProtectedServices[service.Service.ID]
			return fmt.Sprintf("%s:%d", protectedService.Service.Host, protectedService.Service.Port)
		}
	}
	return ""
}
