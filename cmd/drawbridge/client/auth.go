package client

import (
	"cmp"
	"fmt"
	"log/slog"
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
	WANIP        net.IP     `json:"wan-ip"`
	OSType       string     `json:"os-type"`
	SerialNumber string     `json:"serial-number"`
	Operators    []Operator `json:"operators"`
}

// An AuthorizationRequest is all of the characteristics collected about a machine running the Emissary client.
// If an AuthorizationRequest passes the Requirements of an AuthorizationRequest Policy, Drawbridge allow it to access protected resources.
type AuthorizationRequest struct {
	WANIP        net.IP `json:"wan-ip"`
	OSType       string `json:"os-type"`
	SerialNumber string `json:"serial-number"`
}

var TestAuthorizationRequest = AuthorizationRequest{
	WANIP:        net.IPv4(8, 8, 8, 8),
	OSType:       "Windows",
	SerialNumber: "00000",
}

var TestAuthorizationPolicy = AuthorizationPolicy{
	Name:        "Allow Personal Machine",
	Description: "Default policy stuff.",
	Requirements: AuthorizationRequirements{
		WANIP:        net.IPv4(8, 8, 8, 8),
		OSType:       "Windows",
		SerialNumber: "00000",
		Operators:    []Operator{"=", "=", "="},
	},
}

// The Operator is used to evaluate if an Authorization Request field is greater than, is, is not, than an Authorization Requirement value.
type Operator string

func (arv AuthorizationPolicy) ClientIsAuthorized(clientAuthorization AuthorizationRequest) bool {
	authorizationPolicyRequirementsValues := reflect.ValueOf(arv.Requirements)
	clientAuthorizationValues := reflect.ValueOf(clientAuthorization)
	comparisonOperatorValues := arv.Requirements.Operators
	for i := 0; i < clientAuthorizationValues.NumField(); i++ {
		policyValue := authorizationPolicyRequirementsValues.Field(i)
		clientField := clientAuthorizationValues.Field(i)
		comparisonOperator := comparisonOperatorValues[i]
		if !clientAuthorizationFieldMatchesPolicy(clientField.String(), string(comparisonOperator), policyValue.String()) {
			return false
		}
		slog.Debug(fmt.Sprintf("current eval: %s %s %s\n", clientField, comparisonOperator, policyValue))
	}
	return true
}

func clientAuthorizationFieldMatchesPolicy[T cmp.Ordered, V string](fieldValue T, operator V, policyValue T) bool {
	switch operator {
	case "=":
		return fieldValue == policyValue
	case "!=":
		return fieldValue != policyValue
	case ">":
		return fieldValue > policyValue
	case ">=":
		return fieldValue >= policyValue
	case "<":
		return fieldValue < policyValue
	case "<=":
		return fieldValue <= policyValue
	default:
		return false
	}
}
