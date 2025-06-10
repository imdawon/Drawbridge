package drawbridge

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"imdawon/drawbridge/cmd/drawbridge/emissary"
	"imdawon/drawbridge/cmd/drawbridge/persistence"
	"imdawon/drawbridge/cmd/drawbridge/services"
	flagger "imdawon/drawbridge/cmd/flags"
	certificates "imdawon/drawbridge/cmd/reverse_proxy/ca"
	"imdawon/drawbridge/cmd/utils"
	"io"
	"log/slog"
	"math"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
)

type Settings struct {
	ListenerAddress string `schema:"listener-address"`
	EnableDAUPing   bool   `schema:"enable-ping"`
}

// Used by the frontend controller to execute Drawbridge functions.
// ProtectedServices contains a map of listeners running for each Protected Service.
// The int key is the ID of the service as stored in the database.
type Drawbridge struct {
	CA                *certificates.CA
	ProtectedServices map[int64]services.RunningProtectedService
	Settings          *Settings
	DB                *persistence.SQLiteRepository
	ListeningAddress  string
	ListeningPort     uint
	// Contains persistent connections to Emissary Outbound proxy clients, which can expose a service available to it as a Protected Service in Drawbridge.
	OutboundServices map[int64]*services.ProtectedService
	OutboundMutex    sync.RWMutex
}

type EmissaryConfig struct {
	Platform string `schema:"emissary-platform"`
}

// Commented out until we decide to develop device attestation requirements.
//
// When a request comes to our Emissary client api, this function verifies that the body matches the
// Drawbridge Authorization Policy.
// If authorized by passing the policy requirements, we will grant the Emissary client
// an mTLS key to be used by the Emissary client to access an http resource.
// If unauthorized, we send the Emissary client a 401.
// func (d *Drawbridge) handleClientAuthorizationRequest(w http.ResponseWriter, req *http.Request) {
// body, err := io.ReadAll(req.Body)
// if err != nil {
// 	slog.Error("error reading client auth request: %s", err)
// 	w.WriteHeader(http.StatusInternalServerError)
// 	fmt.Fprintf(w, "server error!")
// }

// clientAuth := authorization.EmissaryRequest{}
// err = json.Unmarshal(body, &clientAuth)
// if err != nil {
// 	slog.Error("error unmarshalling client auth request: %s", err)
// 	w.WriteHeader(http.StatusInternalServerError)
// 	fmt.Fprintf(w, "server error!")
// }

// clientIsAuthorized := authorization.TestPolicy.ClientIsAuthorized(clientAuth)
// if clientIsAuthorized {
// 	w.WriteHeader(http.StatusOK)
// 	fmt.Fprintf(w, "client auth success!")
// } else {
// 	w.WriteHeader(http.StatusUnauthorized)
// 	fmt.Fprintf(w, "client auth failure (unauthorized)!")
// }
// }

func (d *Drawbridge) handleEmissaryOutboundRegistration(conn net.Conn, serviceName string) {
	d.OutboundMutex.Lock()
	// TODO - don't pin a set ID, integrate it alongside the other Protected Services.
	d.OutboundServices[999] = &services.ProtectedService{ID: 999, Name: serviceName, Conn: conn}
	d.OutboundMutex.Unlock()

	conn.Write([]byte("ACK"))
	slog.Info("Registered outbound service", slog.String("service", serviceName))
}

// If a Protected Service is being tunneled by an Emissary Outbound client, we have to handle the connection differently than a normal Drawbridge -> Protected Service connection.
// An Emissary Outbound client will send Drawbridge a OB_CR8T string followed by the name of the Protected Service e.g OB_CR8T MyMinecraftServer.
// Once written to Drawbridge, the connection between the Emissary Outbound client and Drawbridge will remain open and Drawbridge will store it in the
// d.OutboundServices map.
// When a regular Emissary client requests to access an Emissary Outbound Protected Service, Drawbridge will get the connection from the OutboundServices map
// mentioned earlier and write all the data the Emissary client sends to the Emissary Outbound Protected service, and vice versa.
func (d *Drawbridge) handleEmissaryOutboundProtectedServiceConnection(emissaryClient net.Conn, serviceName string) {
	defer emissaryClient.Close()

	d.OutboundMutex.RLock()
	outboundService, exists := d.OutboundServices[999]
	d.OutboundMutex.RUnlock()
	if !exists {
		slog.Error("Requested service not found", "serviceName", serviceName)
		return
	} else {
		slog.Debug("Proxying emissary outbound traffic...\n")
	}

	proxyData(outboundService.Conn, emissaryClient)

}

// Set up an mTLS-protected API to serve Emissary client requests.
// The Emissary API is mainly to handle authentication of Emissary clients,
// as well as provisioning mTLS certificates for them.
// Proxying requests for TCP and UDP traffic is handled by the reverse proxy.
func (d *Drawbridge) SetUpEmissaryAPI(hostAndPort string) {
	// r := http.NewServeMux()
	// r.HandleFunc("/emissary/v1/auth", d.handleClientAuthorizationRequest)
	// server := http.Server{
	// 	TLSConfig: d.CA.ServerTLSConfig,
	// 	Addr:      hostAndPort,
	// 	Handler:   r,
	// }
	// slog.Info(fmt.Sprintf("Starting Emissary API server on http://%s", server.Addr))

	// We pass "" into listen and serve since we have already configured cert and keyfile for server.
	// slog.Error("Error starting Emissary API server: %s", server.ListenAndServeTLS("", ""))
}

func (d *Drawbridge) SetUpCAAndDependentServices(protectedServices []services.ProtectedService) {
	certificates.CertificateAuthority = &certificates.CA{DB: d.DB}
	err := certificates.CertificateAuthority.SetupCertificates()
	if err != nil {
		slog.Error("Error setting up root CA: %s", err)
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
func (d *Drawbridge) CreateEmissaryClientTCPMutualTLSKey(clientId, platform string, overrideDirectory ...string) (*string, error) {
	var directoryToSave string
	if len(overrideDirectory) == 0 {
		directoryToSave = "./emissary_certs_and_key_here"
	} else {
		directoryToSave = overrideDirectory[0]
	}
	serverCertExists := utils.FileExists("ca/server-cert.crt")
	if !serverCertExists {
		slog.Error("Unable to create new Emissary Client TCP mTLS key. Server certificate does not exist!")
	}

	listeningAddress, err := d.DB.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		slog.Error("Database", slog.Any("Could not get all services: %s", err))
	}

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
			CommonName:    *listeningAddress,
			SerialNumber:  clientId,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	clientCertPrivKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
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
	err = utils.SaveFile("emissary-mtls-tcp.crt", certPEM.String(), directoryToSave)
	if err != nil {
		return nil, err
	}

	// Android is a special little platform. The Kotlin/Java stdlib seems to only have support for the
	// PKCS8 format. We generate a key in this format for Android to avoid complicated conversion code
	// on the Android client.
	var certPrivKeyPEMBytes []byte
	var certPrivKeyPEM *bytes.Buffer
	if platform == "android" {
		certPrivKeyPEMBytes, err = x509.MarshalPKCS8PrivateKey(clientCertPrivKey)
		if err != nil {
			return nil, err
		}
		certPrivKeyPEM = new(bytes.Buffer)
		pem.Encode(certPrivKeyPEM, &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: certPrivKeyPEMBytes,
		})
		// For non-Android platforms, use the EC Private Key format.
	} else {
		certPrivKeyPEMBytes, err = x509.MarshalECPrivateKey(clientCertPrivKey)
		if err != nil {
			return nil, err
		}
		certPrivKeyPEM = new(bytes.Buffer)
		pem.Encode(certPrivKeyPEM, &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: certPrivKeyPEMBytes,
		})

	}

	// Save the file to disk for use by an Emissary client. This should be later used and saved in the db for downloading later.
	err = utils.SaveFile("emissary-mtls-tcp.key", certPrivKeyPEM.String(), directoryToSave)
	if err != nil {
		slog.Error(fmt.Sprintf("Error saving x509 keypair for Emissary client to disk: %s", err))
	}

	emissaryCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(certPEM.Bytes())
	//  Add Emissary mTLS certificate to list of acceptable client certificates.
	d.CA.ClientTLSConfig.Certificates = append(d.CA.ClientTLSConfig.Certificates, emissaryCert)

	certificateString := certPEM.String()
	hashedCert := certificates.HashEmissaryCertificate(certPEM.Bytes())
	// Add device to certificate list
	d.CA.SetEmissaryCertificateToCertificateList(certPEM.Bytes(), emissary.DeviceCertificate{Revoked: 0, DeviceID: clientId})
	slog.Debug("Certificate List", slog.String("Adding hash", hashedCert))
	slog.Debug("Certificate List", slog.String("plaintext", certPEM.String()))

	return &certificateString, nil
}

var runningProtectedServicesMutex sync.RWMutex

func (d *Drawbridge) AddNewProtectedService(protectedService services.ProtectedService) error {
	runningProtectedServicesMutex.Lock()
	defer runningProtectedServicesMutex.Unlock()
	d.ProtectedServices[protectedService.ID] = services.RunningProtectedService{
		Service: protectedService,
	}
	return nil
}

func (d *Drawbridge) StopRunningProtectedService(id int64) {
	runningProtectedServicesMutex.Lock()
	defer runningProtectedServicesMutex.Unlock()
	delete(d.ProtectedServices, id)
}

// VerifyPeerCertificateWithRevocationCheck is a custom VerifyPeerCertificate callback
// that checks if the peer's certificate is in the revoked certificates list
func (d *Drawbridge) VerifyPeerCertificateWithRevocationCheck(cert string) error {
	d.CA.EmissaryDeviceCertificatesWhitelistMutex.RLock()
	deviceCert, exists := d.CA.GetCertificateFromCertificateList(cert)
	d.CA.EmissaryDeviceCertificatesWhitelistMutex.RUnlock()
	if !exists {
		return errors.New("emissary client presented a certificate that does not exist in our system")
	}
	if deviceCert.Revoked == 1 {
		return errors.New("emissary client presented a certificate that is revoked")
	}
	// If we reach here, no certificate in the chain is revoked
	return nil
}

func proxyData(dst net.Conn, src net.Conn) {
	defer dst.Close()
	defer src.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(dst, src)
		if err != nil {
			slog.Error("Failed to copy src to dst", "error", err)
		}
		dst.Close()

	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(src, dst)
		if err != nil {
			slog.Error("Failed to copy dst to src", "error", err)
		}
		src.Close()
	}()

	wg.Wait()
}

// This is the service the Emissary client connects to when it wants to access a Protected Service.
// It needs to take the Emissary connection and route it to the proper Protected Service.
func (d *Drawbridge) SetUpProtectedServiceTunnel() error {
	// The host and port this tcp server will listen on.
	// This is distinct from the ProtectedService "Host" field, which is the remote address of the actual service itself.
	listeningAddress, err := d.DB.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		slog.Error("Database", slog.Any("Error: %s", err))
	}
	addressAndPort := fmt.Sprintf("%s:%d", *listeningAddress, d.ListeningPort)
	slog.Info(fmt.Sprintf("Starting Drawbridge reverse proxy tunnel. Emissary clients can reach Drawbridge at %s", addressAndPort))
	l, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", d.ListeningPort), d.CA.ServerTLSConfig)
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
		go func(emissaryConn net.Conn) {
			// Read incoming data
			buf := make([]byte, 256)
			n, err := conn.Read(buf)
			if err != nil {
				slog.Error("Protected Service", slog.Any("Connection Read Error: %w", err))
				return
			}
			// Trim unused buffer null terminating characters.
			buf = bytes.Trim(buf, "\x00")
			// Print the incoming data - for debugging
			slog.Info("Emissary Connection", slog.Any("Message Received", buf))

			emissaryRequestPayload := string(buf[:n])
			emissaryRequestType := emissaryRequestPayload[:7]
			emissaryRequestedServiceId := ""
			if emissaryRequestType != "PS_LIST" {
				emissaryRequestedServiceId = emissaryRequestPayload[8:11]
			}

			// Create and insert an Emissary event into the db.
			eventUUID, err := utils.NewUUID()
			if err != nil {
				slog.Error("Emissary Event", slog.Any("Error", err))
			}
			// Retrieve the client certificate from the connection by casting the connection as a tls.Conn.
			clientCert := emissaryConn.(*tls.Conn).ConnectionState().PeerCertificates[0]
			deviceUUID := clientCert.Subject.SerialNumber
			event := emissary.Event{
				ID:             eventUUID,
				DeviceID:       deviceUUID,
				ConnectionIP:   conn.RemoteAddr().String(),
				Type:           emissaryRequestType,
				TargetService:  emissaryRequestedServiceId,
				ConnectionType: "",
				Timestamp:      time.Now().Format(time.RFC3339),
			}
			go func() {
				slog.Debug("Inserting Emissary Event...")
				err = d.DB.InsertEmissaryClientEvent(event)
			}()
			if err != nil {
				slog.Error("Emissary Event", slog.Any("DB Error", err))
			}

			switch emissaryRequestType {
			case "OB_CR8T":
				slog.Debug("Create Outbound Protected Service Request - handling...")
				d.handleEmissaryOutboundRegistration(conn, emissaryRequestPayload[18:])
			case "PS_CONN":
				// May be used later after we standardize how and when to read the tcp connection into the buf above.
				// d.getRequestProtectedServiceName(clientConn)
				emissaryRequestedServiceIdNum, err := strconv.Atoi(emissaryRequestedServiceId)
				if err != nil {
					slog.Error("PS_CONN Handler", slog.Any("Error converting first byte of emissary request service id to int", err))
				}
				requestedServiceAddress, tunnelType := d.getProtectedServiceAddressById(emissaryRequestedServiceIdNum)
				// For Emissary OB (Outbound) connects, Drawbridge will actually connect to an Emissary client which is exposing a
				// locally accessible network service.
				if tunnelType == "OB" {
					slog.Debug("Outbound Protected Service Detected - handling connection...")
					d.handleEmissaryOutboundProtectedServiceConnection(emissaryConn, requestedServiceAddress)
					break
				}

				// Proxy traffic to the actual service the Emissary client is trying to connect to.
				var dialer net.Dialer
				var protectedServiceConn net.Conn
				const maxRetries = 5
				const baseDelay = 500 * time.Millisecond
				const maxDelay = 16 * time.Second

				for retries := 0; retries < maxRetries; retries++ {
					protectedServiceConn, err = establishConnection(dialer, requestedServiceAddress)
					if err == nil {
						// Connection established successfully, handle it
						break
					}

					// Calculate delay with exponential backoff
					delay := time.Duration(math.Pow(2, float64(retries))) * baseDelay
					if delay > maxDelay {
						delay = maxDelay
					}

					// Add jitter
					jitterMax := big.NewInt(int64(float64(delay) * 0.1))
					jitterInt, _ := rand.Int(rand.Reader, jitterMax)
					jitter := time.Duration(jitterInt.Int64())
					delay += jitter

					slog.Error("Failed to establish connection to Protected Service. Retrying...",
						"error", err,
						"retryCount", retries+1,
						"nextRetryIn", delay)

					time.Sleep(delay)
				}

				if err != nil {
					slog.Error("Failed to establish connection after max retries",
						"maxRetries", maxRetries,
						"error", err)
					return
				}

				// Connection established successfully, continue with handling...
				// This can happen if the Drawbridge admin deletes a Protected Service while it is running.
				// The net.Listener will be closed and any remaining Accept operations are blocked and return errors.
				if err != nil {
					slog.Error("Failed to tcp dial to actual target service", err)
				}

				slog.Debug(fmt.Sprintf("TCP Accept from Emissary client: %s", emissaryConn.RemoteAddr()))
				// Copy data back and from client and server.
				proxyData(protectedServiceConn, emissaryConn)
				// Shut down the connection.
				emissaryConn.Close()
			case "PS_LIST":
				// On a new connection, write available services to TCP connection so Emissary can know which
				// Protected Services are available
				var serviceList string
				for _, value := range d.ProtectedServices {
					// We pad the service id with zeros as we want a fixed-width id for easy parsing. This will allow support for up to 1000 Protected Services.
					serviceList += fmt.Sprintf("%s%s,", utils.PadWithZeros(int(value.Service.ID)), value.Service.Name)
				}
				for _, value := range d.OutboundServices {
					// We pad the service id with zeros as we want a fixed-width id for easy parsing. This will allow support for up to 1000 Protected Services.
					serviceList += fmt.Sprintf("%s%s,", utils.PadWithZeros(int(value.ID)), value.Name)
				}
				// The newline character is important for other platforms, such as Android,
				// to properly read the string from the socket without blocking.
				serviceConnectCommand := fmt.Sprintf("PS_LIST: %s\n", serviceList)
				slog.Debug(fmt.Sprintf("PS_LIST values: %s\n", serviceConnectCommand))
				emissaryConn.Write([]byte(serviceConnectCommand))
			default:
			}
		}(conn)
	}
}

func establishConnection(dialer net.Dialer, serviceAddress string) (net.Conn, error) {
	resourceConn, err := dialer.Dial("tcp", serviceAddress)
	if err == nil {
		return resourceConn, nil
	}
	return nil, err

}

func (d *Drawbridge) getRequestProtectedServiceName(clientConn net.Conn) (string, error) {
	bytes, err := io.ReadAll(io.LimitReader(clientConn, 64))
	if err != nil {
		return "", err
	}

	return string(bytes[:]), nil
}

// Returns the service key (id) and the service type "PS" for a regular Protected Service
// and "OB" for an Emissary Outbound Protected Service.
func (d *Drawbridge) getProtectedServiceAddressById(protectedServiceId int) (string, string) {
	for _, service := range d.ProtectedServices {
		if service.Service.ID == int64(protectedServiceId) {
			protectedService := d.ProtectedServices[service.Service.ID]
			return fmt.Sprintf("%s:%d", protectedService.Service.Host, protectedService.Service.Port), "PS"
		}
	}
	for _, outboundService := range d.OutboundServices {
		if outboundService.ID == int64(protectedServiceId) {
			protectedService := d.ProtectedServices[outboundService.ID]
			return fmt.Sprintf("%s:%d", protectedService.Service.Host, protectedService.Service.Port), "OB"
		}
	}

	slog.Error("Unable to find service id mapping for id", slog.Int("protectedServiceId", protectedServiceId))

	return "", ""
}

type GitHubLatestReleaseBody struct {
	AssetsURL string `json:"assets_url"`
}

type GitHubLatestAssetsBody struct {
	Asset string `json:"browser_download_url"`
	Name  string `json:"name"`
}

type BundleFile struct {
	Contents *[]byte
	Name     string
}

// * This is a very important / dangerous function *
// A Drawbridge admin can generate an "Emissary Bundle" which adds
// the encryption keys, certs, and drawbridge connection address alongside the Emissary client binary.
// This reduces the need for an Emissary user to manually configure the Emissary client at all.
// To accomplish this, we pull the latest version of Emissary from GitHub Releases, verify it is signed with the
// Drawbridge & Emissary signing key, generate the mTLS key(s) and cert, zip it all up, and allow the Drawbridge admin to download it.
func (d *Drawbridge) GenerateEmissaryBundle(config EmissaryConfig) (*BundleFile, error) {
	if config.Platform != "macos" && config.Platform != "linux" && config.Platform != "windows" && config.Platform != "android" {
		return nil, fmt.Errorf("platform %s is not supported", config.Platform)
	}

	if config.Platform == "android" || config.Platform == "ios" {
		slog.Debug("Making mobile platform Emissary Bundle")
		return d.generateMobileEmissaryBundle(config.Platform)
	}

	// Get assets url
	releaseResp, err := http.Get("https://api.github.com/repos/imdawon/Emissary-Daemon/releases/latest")
	if err != nil {
		return nil, err
	}
	releaseBody, err := io.ReadAll(io.LimitReader(releaseResp.Body, 500000))
	if err != nil {
		return nil, err
	}

	var githubReleaseBody GitHubLatestReleaseBody
	json.Unmarshal(releaseBody, &githubReleaseBody)
	// Ensure we only allow legit URLs in case the response gets hijacked / modified somehow.
	// We don't want make a request get whatever arbitrary response url is returned from the GitHub API.
	if githubReleaseBody.AssetsURL[:62] != "https://api.github.com/repos/imdawon/Emissary-Daemon/releases/" {
		return nil, fmt.Errorf("unexpected url returned from github 'releases/latest' endpoint. unable to get Emissary client")
	}

	// Get all asset file metadata for latest release
	assetsResp, err := http.Get(githubReleaseBody.AssetsURL)
	if err != nil {
		return nil, err
	}
	assetsBody, err := io.ReadAll(io.LimitReader(assetsResp.Body, 500000))
	if err != nil {
		return nil, err
	}

	var githubAssetsBody []GitHubLatestAssetsBody
	json.Unmarshal(assetsBody, &githubAssetsBody)
	var emissaryClientURL string
	var emissaryClientSigURL string
	var emissaryClientFilename string
	var emissaryClientSigFilename string
	// Ensure we only allow legit URLs in case the response gets hijacked / modified somehow.
	// We don't want make a request get whatever arbitrary response url is returned from the GitHub API.
	for _, v := range githubAssetsBody {
		if len(emissaryClientURL) > 0 && len(emissaryClientSigURL) > 0 {
			break
		}
		assetURL := v.Asset
		if v.Asset[:61] != "https://github.com/imdawon/Emissary-Daemon/releases/download/" {
			return nil, fmt.Errorf("unexpected url returned from github 'releases/latest' endpoint. unable to get Emissary client")
		}
		// Add all macos asset files since we need the zipped Emissary client and the .sig file.
		if strings.Contains(assetURL, config.Platform) {
			if strings.Contains(assetURL, "asc") {
				emissaryClientSigURL = assetURL
				emissaryClientSigFilename = v.Name
				continue
			}
			emissaryClientFilename = v.Name
			emissaryClientURL = assetURL
		}
	}

	// Grab the latest Emissary release (macOS, Linux, or Windows) GitHub Releases API
	emissaryResp, err := http.Get(emissaryClientURL)
	if err != nil {
		return nil, err
	}
	// We don't expect the zipped Emissary Bundle to get larger than 10MB
	emissaryReleaseBody, err := io.ReadAll(io.LimitReader(emissaryResp.Body, 10000000))
	if err != nil {
		return nil, err
	}

	var githubEmissaryReleaseBody GitHubLatestReleaseBody
	json.Unmarshal(emissaryReleaseBody, &githubEmissaryReleaseBody)

	// Grab the latest Emissary release (macOS, Linux, or Windows) signature file from GitHub Releases API
	emissarySigResp, err := http.Get(emissaryClientSigURL)
	if err != nil {
		return nil, err
	}
	emissarySigBody, err := io.ReadAll(io.LimitReader(emissarySigResp.Body, 500))
	if err != nil {
		return nil, err
	}

	var githubEmissarySigBody GitHubLatestReleaseBody
	json.Unmarshal(emissarySigBody, &githubEmissarySigBody)

	// Verify the Emissary file we downloaded is properly signed with the Drawbridge & Emissary Signing Key.
	pgp := crypto.PGP()
	publicKey, err := crypto.NewKeyFromArmored(DRAWBRIDGE_AND_EMISSARY_SIGNING_PUBKEY)
	if err != nil {
		return nil, err
	}
	verifier, err := pgp.Verify().VerificationKey(publicKey).New()
	if err != nil {
		return nil, err
	}
	verifyResult, err := verifier.VerifyDetached(emissaryReleaseBody, emissarySigBody, crypto.Armor)
	if err != nil {
		slog.Error("Emissary Bundle Creation", slog.Any("Internal Non-signature error when attempting to validate Emissary .zip file against .asc file", err))

		return nil, fmt.Errorf("err verifying dettached: %w", err)
	}
	// If this fails that means the Emissary client we downloaded is not properly signed and may have been tampered with.
	// In this situation, we cannot continue this process and must abort.
	if sigErr := verifyResult.SignatureError(); sigErr != nil {
		slog.Error("Emissary Bundle Creation", slog.Any("Error", fmt.Errorf("POTENTIAL DANGER!!! Error verifying authenticity of signed Emissary .zip file! Someone could be attempting to serve a malicious copy of Emissary, or the Emissary file was corrupted during download from GitHub: %w", err)))
		return nil, fmt.Errorf("emissary signature error: %w", sigErr)
	}

	// We don't care that we are modifying these files and sending them to the client without re-signing.
	// The client isn't supposed to do any manual config anyway.
	// For power-users, we could re-sign our Emissary Bundle with their Drawbridge CA.
	slog.Debug("verified file")
	// Save Emissary .zip file contents to disk
	bundleTmpFolderPath := "./bundle_tmp"
	// Create temporary directory used for placing Emissary files to zip up for use as the downloadable Emissary Bundle.
	os.Mkdir(utils.CreateDrawbridgeFilePath(bundleTmpFolderPath), os.ModePerm)
	emissaryDownloadFolder := "./emissary_download_scratch"
	utils.SaveFileByte(emissaryClientFilename, emissaryReleaseBody, emissaryDownloadFolder)
	// Save Emissary .zip .asc file contents to disk
	utils.SaveFileByte(emissaryClientSigFilename, emissarySigBody, emissaryDownloadFolder)
	slog.Debug("saved emissary and sig files")

	// Unzip the Emissary release
	// TODO
	// create bundle_tmp folder before running this.
	// ./emissary_download_scratch/Emissary_platform_xxx.zip
	emissaryZipPath := path.Join(emissaryDownloadFolder, emissaryClientFilename)
	// Unzip the zip file into the bundle_tmp directory.
	// We will be zipping up the contents of the ./bundle_tmp directory later.
	slog.Debug("unzipping emissary zip file from github...", slog.Any("Path", emissaryZipPath))
	err = utils.Unzip(emissaryZipPath, bundleTmpFolderPath)
	if err != nil {
		slog.Error("Emissary Bundle Creation", slog.Any("Error", fmt.Errorf("unable to unzip Emissary client downloaded from GitHub: %w", err)))
		return nil, err
	}
	// Generate and save the mTLS key(s) and cert
	clientId, err := utils.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error generating uuid: %w", err)
	}
	certsAndKeysFolderPath := "./bundle_tmp/put_certificates_and_key_from_drawbridge_here"
	emissaryCert, err := d.CreateEmissaryClientTCPMutualTLSKey(clientId, config.Platform, certsAndKeysFolderPath)
	if err != nil {
		return nil, err
	}
	// Copy ca.crt next to keys
	err = utils.CopyFile("./ca/ca.crt", certsAndKeysFolderPath)
	if err != nil {
		slog.Error("Emissary Bundle Creation", slog.Any("Error", fmt.Errorf("unable to copy the Drawbridge ca.crt file to the Emissary Bundle put_certificates_... folder: %s", err)))
		return nil, err
	}
	// Generate and save bundle using Drawbridge listening address
	listeningAddress, err := d.DB.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		return nil, err
	}
	if len(*listeningAddress) > 0 {
		// TODO
		// Change the port hardcoding and write the listening port in the lsiteningAddress config file instead.
		utils.SaveFile("drawbridge.txt", fmt.Sprintf("%s:%d", *listeningAddress, d.ListeningPort), "./bundle_tmp/bundle")
	} else {
		slog.Error("Emissary Bundle Creation", slog.String("Error", "Unable to get Drawbridge listening address. Unable to finish creating bundle."))
		return nil, fmt.Errorf("error getting Drawbridge listening address")
	}
	// Zip up Emissary directory to bundles output folder.
	bundledFilename := fmt.Sprintf("./bundled_%s_%s", clientId, emissaryClientFilename)
	// TODO
	// return the file contents rather than writing to disk by default.
	// there are tons of situations where we'd prefer to just hand off the bytes to the Drawbridge admin in the
	// form of a file.
	utils.ZipSource(bundleTmpFolderPath, bundledFilename)

	// Serve to Drawbridge admin
	slog.Debug("reading bundledemissary output file to send back to admin...")
	bundledEmissaryZipFile := utils.ReadFile(bundledFilename)
	// Remove temp folders
	defer os.RemoveAll("./bundle_tmp")
	defer os.RemoveAll("./emissary_download_scratch")
	bundleFile := BundleFile{
		Contents: bundledEmissaryZipFile,
		Name:     bundledFilename,
	}

	err = d.createEmissaryDevice(clientId, *emissaryCert)
	if err != nil {
		return nil, err
	}

	return &bundleFile, nil
}

func (d *Drawbridge) createEmissaryDevice(id, certificate string) error {
	adjectivesIndex := utils.RandInt(0, len(Adjectives))
	animalsIndex := utils.RandInt(0, len(Animals))
	deviceName := fmt.Sprintf("%s %s", Adjectives[adjectivesIndex], Animals[animalsIndex])

	client := emissary.EmissaryClient{
		ID:                    id,
		Name:                  deviceName,
		DrawbridgeCertificate: certificate,
		Revoked:               0,
	}
	_, err := d.DB.CreateNewEmissaryClient(client)
	if err != nil {
		return fmt.Errorf("error creating emissary client: %w", err)
	}
	return nil
}

// Generate an Emissary Bundle for a mobile device.
// We can't fling .apk or .ipa files at mobile users, so we instead just ship the bundle with our certs, keypair, and drawbridge address.
func (d *Drawbridge) generateMobileEmissaryBundle(platform string) (*BundleFile, error) {
	bundleTmpFolderPath := "./bundle_tmp"
	// Create temporary directory used for placing Emissary files to zip up for use as the downloadable Emissary Bundle.
	os.Mkdir(utils.CreateDrawbridgeFilePath(bundleTmpFolderPath), os.ModePerm)

	// Generate and save the mTLS key(s) and cert
	clientId, err := utils.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error generating uuid: %w", err)
	}
	certsAndKeysFolderPath := "./bundle_tmp/put_certificates_and_key_from_drawbridge_here"
	emissaryCert, err := d.CreateEmissaryClientTCPMutualTLSKey(clientId, platform, certsAndKeysFolderPath)
	if err != nil {
		return nil, err
	}
	// Copy ca.crt next to keys
	err = utils.CopyFile("./ca/ca.crt", certsAndKeysFolderPath)
	if err != nil {
		slog.Error("Emissary Bundle Creation", slog.Any("Error", fmt.Errorf("unable to copy the Drawbridge ca.crt file to the Emissary Bundle put_certificates_... folder: %s", err)))
		return nil, err
	}
	// Generate and save bundle using Drawbridge listening address
	listeningAddress, err := d.DB.GetDrawbridgeConfigValueByName("listening_address")
	if err != nil {
		return nil, err
	}
	if len(*listeningAddress) > 0 {
		// TODO
		// Change the port hardcoding and write the listening port in the lsiteningAddress config file instead.
		utils.SaveFile("drawbridge.txt", fmt.Sprintf("%s:%d", *listeningAddress, d.ListeningPort), "./bundle_tmp/bundle")
	} else {
		slog.Error("Emissary Bundle Creation", slog.String("Error", "Unable to get Drawbridge listening address. Unable to finish creating bundle."))
		return nil, fmt.Errorf("error getting Drawbridge listening address")
	}
	// Zip up Emissary directory to bundles output folder.
	bundledFilename := fmt.Sprintf("./android_bundle_%s", clientId)
	// TODO
	// return the file contents rather than writing to disk by default.
	// there are tons of situations where we'd prefer to just hand off the bytes to the Drawbridge admin in the
	// form of a file.
	utils.ZipSource(bundleTmpFolderPath, bundledFilename)

	// Serve to Drawbridge admin
	slog.Debug("reading bundled emissary output file to send back to admin...")
	bundledEmissaryZipFile := utils.ReadFile(bundledFilename)
	// Remove temp folders
	defer os.RemoveAll("./bundle_tmp")
	defer os.RemoveAll("./emissary_download_scratch")
	bundleFile := BundleFile{
		Contents: bundledEmissaryZipFile,
		Name:     bundledFilename,
	}

	err = d.createEmissaryDevice(clientId, *emissaryCert)
	if err != nil {
		return nil, err
	}
	return &bundleFile, nil

}
