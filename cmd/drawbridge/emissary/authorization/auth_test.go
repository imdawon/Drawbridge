package authorization

import (
	"net"
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

// TestClientIsAuthorizedDenied tests authorization when request doesn't match policy
func TestClientIsAuthorizedDenied(t *testing.T) {
	// Test policy defined but not used in this test
	
	// Create a request that doesn't match policy requirements
	mismatchedRequest := EmissaryRequest{
		WANIP:        net.IPv4(1, 1, 1, 1), // Different IP
		OSType:       "Windows",            // Same OS
		SerialNumber: "00000",              // Same Serial
	}
	
	// Create a policy with different IP requirement
	customPolicy := Policy{
		Name:        "Custom Policy",
		Description: "Policy that should reject this request",
		Requirements: Requirements{
			WANIP:        net.IPv4(9, 9, 9, 9), // Different from request
			OSType:       "Windows",
			SerialNumber: "00000",
			Operators:    []Operator{"!=", "=", "="},
		},
	}
	
	// Should be denied
	got := customPolicy.ClientIsAuthorized(mismatchedRequest)
	if got {
		t.Errorf("client authorization: got %t; wanted %t", got, false)
	}
}

// TestOperatorComparisons tests the different operator comparisons
func TestOperatorComparisons(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		a        string
		b        string
		expected bool
	}{
		{"Equal true", "=", "test", "test", true},
		{"Equal false", "=", "test", "different", false},
		{"Not equal true", "!=", "test", "different", true},
		{"Not equal false", "!=", "test", "test", false},
		{"Greater than true", ">", "b", "a", true},
		{"Greater than false", ">", "a", "b", false},
		{"Less than true", "<", "a", "b", true},
		{"Less than false", "<", "b", "a", false},
		{"Greater than or equal true (equal)", ">=", "a", "a", true},
		{"Greater than or equal true (greater)", ">=", "b", "a", true},
		{"Greater than or equal false", ">=", "a", "b", false},
		{"Less than or equal true (equal)", "<=", "a", "a", true},
		{"Less than or equal true (less)", "<=", "a", "b", true},
		{"Less than or equal false", "<=", "b", "a", false},
		{"Invalid operator", "invalid", "a", "b", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clientAuthorizationFieldMatchesPolicy(tt.a, tt.operator, tt.b)
			if result != tt.expected {
				t.Errorf("clientAuthorizationFieldMatchesPolicy(%s, %s, %s) = %v; want %v", 
					tt.a, tt.operator, tt.b, result, tt.expected)
			}
		})
	}
}
