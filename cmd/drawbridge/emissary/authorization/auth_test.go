package authorization

import (
	"testing"
)

func TestClientIsAuthorized(t *testing.T) {
	arv := TestPolicy
	testAuthorizationRequest := ValidateEmissaryRequest
	got := arv.ClientIsAuthorized(testAuthorizationRequest)
	if !got {
		t.Errorf("client is authorized %t; wanted %t", got, true)
	}
}
