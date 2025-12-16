package envoy

import (
	"fmt"
	"os/exec"
)

// Validator validates Envoy configuration files
type Validator struct {
	envoyBinary string
}

// NewValidator creates a new Envoy config validator
func NewValidator(envoyBinary string) *Validator {
	return &Validator{
		envoyBinary: envoyBinary,
	}
}

// ValidateConfig validates an Envoy configuration file
func (v *Validator) ValidateConfig(configPath string) error {
	// Run envoy with --mode validate
	// #nosec G204 -- envoyBinary is set at initialization, not from user input
	cmd := exec.Command(v.envoyBinary, "--mode", "validate", "-c", configPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// ValidateBootstrap validates the bootstrap configuration
func (v *Validator) ValidateBootstrap(bootstrapPath string) error {
	return v.ValidateConfig(bootstrapPath)
}
