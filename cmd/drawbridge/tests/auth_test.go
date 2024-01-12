package tests

import (
	auth "dhens/drawbridge/cmd/drawbridge/client"
	"testing"
)

func TestClientIsAuthorized(t *testing.T) {
	arv := auth.TestAuthorizationPolicy
	testAuthorizationRequest := auth.TestAuthorizationRequest
	got := arv.ClientIsAuthorized(testAuthorizationRequest)
	if !got {
		t.Errorf("client is authorized %t; wanted %t", got, true)
	}
}
