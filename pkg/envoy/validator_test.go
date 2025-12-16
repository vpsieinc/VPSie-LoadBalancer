package envoy

import "testing"

func TestNewValidator(t *testing.T) {
	validator := NewValidator("/usr/bin/envoy")

	if validator.envoyBinary != "/usr/bin/envoy" {
		t.Errorf("envoyBinary = %v, want /usr/bin/envoy", validator.envoyBinary)
	}
}

func TestValidator_ValidateBootstrap(t *testing.T) {
	// This test verifies that ValidateBootstrap calls ValidateConfig
	// We can't test the actual validation without envoy binary installed
	validator := NewValidator("/nonexistent/envoy")

	err := validator.ValidateBootstrap("/path/to/config.yaml")

	// Should error because envoy binary doesn't exist
	if err == nil {
		t.Error("Expected error when envoy binary doesn't exist")
	}
}
