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
func SetUpProtectedServiceTunnel(protectedService *drawbridge.ProtectedService, ca *proxy.CA) {
	// The host and port this tcp server will listen on.
	// This is distinct from the ProtectedService "Host" field, which is the remote address of the actual service itself.
	hostAndPort := fmt.Sprintf("%s:%d", "localhost", protectedService.Port)
	slog.Info(fmt.Sprintf("Spinning up TCP Listener for Protected Service \"%s\" on %s", protectedService.Name, hostAndPort))
	l, err := tls.Listen("tcp", hostAndPort, ca.ServerTLSConfig)
	if err != nil {
		slog.Error(fmt.Sprintf("TCP Listen failed: %s", err))
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

			// connect to drawbridge on the port lsitening for the actual service
			resourceConn, err := d.DialContext(ctx, "tcp", hostAndPort)
			if err != nil {
				log.Fatalf("Failed to tcp dial to drawbridge server: %v", err)
			}

			slog.Info("TCP Accept from: %s\n", clientConn.RemoteAddr())
			// Copy data back and from client and server.
			go io.Copy(resourceConn, clientConn)
			io.Copy(clientConn, resourceConn)
			// Shut down the connection.
			clientConn.Close()
		}(conn)
	}
}
