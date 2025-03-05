package certificates

import (
	"dhens/drawbridge/cmd/drawbridge/emissary"
	"net"
	"testing"
)

// TestCASetupCertificates tests the certificate generation process
// This is a simplified test that verifies basic functionality
func TestDrawbridgeListeningAddressIsLAN(t *testing.T) {
	tests := []struct {
		name     string
		ip       net.IP
		expected bool
	}{
		{"Local loopback", net.ParseIP("127.0.0.1"), true},
		{"Class A private", net.ParseIP("10.0.0.1"), true},
		{"Class B private", net.ParseIP("172.16.0.1"), true},
		{"Class C private", net.ParseIP("192.168.1.1"), true},
		{"Public IP", net.ParseIP("8.8.8.8"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := drawbridgeListeningAddressIsLAN(tt.ip)
			if result != tt.expected {
				t.Errorf("drawbridgeListeningAddressIsLAN(%v) = %v; want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestHashEmissaryCertificate tests the certificate hashing function
func TestHashEmissaryCertificate(t *testing.T) {
	// Create a simple test certificate
	cert := []byte{1, 2, 3, 4, 5}
	
	// Hash should be deterministic for the same input
	hash1 := hashEmissaryCertificate(cert)
	hash2 := hashEmissaryCertificate(cert)
	
	if hash1 != hash2 {
		t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
	}
	
	// Different certificates should produce different hashes
	differentCert := []byte{5, 4, 3, 2, 1}
	hash3 := hashEmissaryCertificate(differentCert)
	
	if hash1 == hash3 {
		t.Errorf("different certificates produced the same hash: %s", hash1)
	}
}

// TestCertificateRevocation tests the certificate revocation functionality
func TestCertificateRevocation(t *testing.T) {
	ca := &CA{
		CertificateList: make(map[string]emissary.DeviceCertificate),
	}
	
	// Add a test certificate
	testCertHash := "testhash123"
	ca.CertificateList[testCertHash] = emissary.DeviceCertificate{
		DeviceID: "test-device",
		Revoked:  0,
	}
	
	// Test revocation
	ca.RevokeCertInCertificateRevocationList(testCertHash)
	if ca.CertificateList[testCertHash].Revoked != 1 {
		t.Errorf("certificate not properly revoked")
	}
	
	// Test unrevocation
	ca.UnRevokeCertInCertificateRevocationList(testCertHash)
	if ca.CertificateList[testCertHash].Revoked != 0 {
		t.Errorf("certificate not properly unrevoked")
	}
	
	// Test revocation of non-existent certificate
	nonExistentHash := "nonexistent"
	ca.RevokeCertInCertificateRevocationList(nonExistentHash)
	// Should not panic or add the certificate
	if _, exists := ca.CertificateList[nonExistentHash]; exists {
		t.Errorf("non-existent certificate was added to the list")
	}
}

// TestVerifyEmissaryCertificate tests the certificate verification function
func TestVerifyEmissaryCertificate(t *testing.T) {
	ca := &CA{
		CertificateList: make(map[string]emissary.DeviceCertificate),
	}
	
	// Add a valid certificate
	validCert := []byte{1, 2, 3, 4, 5}
	validHash := hashEmissaryCertificate(validCert)
	ca.CertificateList[validHash] = emissary.DeviceCertificate{
		DeviceID: "valid-device",
		Revoked:  0,
	}
	
	// Add a revoked certificate
	revokedCert := []byte{6, 7, 8, 9, 10}
	revokedHash := hashEmissaryCertificate(revokedCert)
	ca.CertificateList[revokedHash] = emissary.DeviceCertificate{
		DeviceID: "revoked-device",
		Revoked:  1,
	}
	
	// Test with empty certificate list
	err := ca.verifyEmissaryCertificate([][]byte{}, nil)
	if err == nil {
		t.Errorf("empty certificate list should return an error")
	}
	
	// Test with valid certificate
	err = ca.verifyEmissaryCertificate([][]byte{validCert}, nil)
	if err != nil {
		t.Errorf("valid certificate verification failed: %v", err)
	}
	
	// Test with revoked certificate
	err = ca.verifyEmissaryCertificate([][]byte{revokedCert}, nil)
	if err == nil {
		t.Errorf("revoked certificate verification should fail")
	}
	
	// Test with unknown certificate
	unknownCert := []byte{11, 12, 13, 14, 15}
	err = ca.verifyEmissaryCertificate([][]byte{unknownCert}, nil)
	if err == nil {
		t.Errorf("unknown certificate verification should fail")
	}
}