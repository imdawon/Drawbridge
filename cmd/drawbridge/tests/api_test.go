package tests

import (
	"testing"
)

func testMakeHttpClient(t *testing.T) {
	// transport := &http.Transport{
	// 	TLSClientConfig: adminAPI.CA.ClientTLSConfig,
	// }
	// return &http.Client{
	// 	Transport: transport,
	// }
}

func testMakeClientAuthorizationRequest() {
	// Communicate with the http server using an http.Client configured to trust our Drawbridge CA.
	// transport := &http.Transport{
	// 	TLSClientConfig: adminAPI.CA.ClientTLSConfig,
	// }
	// http := http.Client{
	// 	Transport: transport,
	// }

	// authorizationRequest := emissary.TestAuthorizationRequest
	// out, err := json.Marshal(authorizationRequest)
	// if err != nil {
	// 	log.Fatalf("failed to marshal auth request: %s", err)
	// }

	// resp, err := http.Post("https://localhost:3001/emissary/v1/auth", "application/json", bytes.NewBuffer(out))
	// if err != nil {
	// 	log.Fatalf("POST to auth endpoint failed: %s", err)
	// }
	// respBodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	slog.Error("Error reading body response from client auth request: %s", err)
	// }
	// body := strings.TrimSpace(string(respBodyBytes[:]))
	// slog.Debug(fmt.Sprintf("client request body: %s", body))
}

func testMakeClientHttpRequest(url string) {
	// Communicate with the http server using an http.Client configured to trust our CA.
	// resp, err := http.Get(url)
	// if err != nil {
	// 	log.Fatalf("GET request to %s failed: %s", url, err)
	// }
	// respBodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Printf("Error reading body response from reverse proxy request: %s", err)
	// }
	// body := strings.TrimSpace(string(respBodyBytes[:]))
	// slog.Info(fmt.Sprintf("client request body: %s", body))
}
