package proxy

import (
	"context"
	"crypto/tls"
	"dhens/drawbridge/cmd/drawbridge"
	proxy "dhens/drawbridge/cmd/reverse_proxy/ca"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"time"
)

// This function will be called Whenever a new Protected Service is:
// 1. Created in the dash
// 2. Loaded from disk during Drawbridge startup
// This function call sets up a tcp server, and each Protected Service gets its own tcp server.
func SetUpProtectedServiceTunnel(protectedService drawbridge.ProtectedService, ca *proxy.CA) {
	// The host and port this tcp server will listen on.
	// This is distinct from the ProtectedService "Host" field, which is the remote address of the actual service itself.
	slog.Info(fmt.Sprintf("Starting tunnel for Protected Service \"%s\". Emissary clients can reach this service at %s", protectedService.Name, "0.0.0.0:3100"))
	l, err := tls.Listen("tcp", "0.0.0.0:3100", ca.ServerTLSConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Reverse proxy TCP Listen failed: %s", err))
	}

	defer l.Close()
	for {
		// wait for connection
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Reverse proxy TCP Accept failed: %s", err)
		}
		// Handle new connection in a new go routine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(clientConn net.Conn) {
			var d net.Dialer
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			// Proxy traffic to the actual service the Emissary client is trying to connect to.
			resourceConn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", protectedService.Host, protectedService.Port))
			if err != nil {
				log.Fatalf("Failed to tcp dial to actual target serviec: %v", err)
			}

			slog.Info(fmt.Sprintf("TCP Accept from Emissary client: %s\n", clientConn.RemoteAddr()))
			// Copy data back and from client and server.
			go io.Copy(resourceConn, clientConn)
			io.Copy(clientConn, resourceConn)
			// Shut down the connection.
			clientConn.Close()
		}(conn)
	}
}
