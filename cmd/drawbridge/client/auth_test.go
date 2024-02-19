package client

import (
	"testing"
)

func TestClientIsAuthorized(t *testing.T) {
	arv := TestAuthorizationPolicy
	testAuthorizationRequest := TestAuthorizationRequest
	got := arv.ClientIsAuthorized(testAuthorizationRequest)
	if !got {
		t.Errorf("client is authorized %t; wanted %t", got, true)
	}
}
