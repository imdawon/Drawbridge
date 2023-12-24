package auth

import (
	"fmt"
	"net"
	"reflect"
)

// The policy that Drawbridge will use to evaluate if it will allow access to an Emissary client.
type AuthorizationPolicy struct {
	ID           int64
	Name         string                    `schema:"policy-name" json:"policy-name"`
	Description  string                    `schema:"policy-description" json:"policy-description"`
	Requirements AuthorizationRequirements `schema:"policy-requirements" json:"policy-requirements"`
}

// The actual required values of each field
type AuthorizationRequirements struct {
	WANIP                net.IP   `json:"wan-ip"`
	WANIPOperator        Operator `json:"wan-ip-operator"`
	OSType               string   `json:"os-type"`
	OSTypeOperator       Operator `json:"os-type-operator"`
	SerialNumber         string   `json:"serial-number"`
	SerialNumberOperator Operator `json:"serial-number-operator"`
}

// An AuthorizationRequest is all of the characteristics collected about a machine running the Emissary client.
// If an AuthorizationRequest passes the Requirements of an AuthorizationRequest Policy, Drawbridge allow it to access protected resources.
type AuthorizationRequest struct {
	WANIP        net.IP `json:"wan-ip"`
	OSType       string `json:"os-type"`
	SerialNumber string `json:"serial-number"`
}

var TestAuthorizationPolicy = AuthorizationPolicy{
	Name:        "Allow Personal Machine",
	Description: "",
	Requirements: AuthorizationRequirements{
		WANIP:                net.IPv4(8, 8, 8, 8),
		WANIPOperator:        "=",
		OSType:               "Windows",
		OSTypeOperator:       "=",
		SerialNumber:         "00000",
		SerialNumberOperator: "=",
	},
}

// The Operator is used to evaluate if an Authorization Request field is greater than, is, is not, than an Authorization Requirement value.
type Operator string

func (arv *AuthorizationPolicy) clientIsAuthorized(clientAuthorization AuthorizationRequest) bool {
	authorizationPolicyRequirements := reflect.ValueOf(arv.Requirements)
	for i := 0; i < authorizationPolicyRequirements.NumField(); i++ {
		fmt.Printf("value: %v", authorizationPolicyRequirements.Field(i))

	}
	return true
}
