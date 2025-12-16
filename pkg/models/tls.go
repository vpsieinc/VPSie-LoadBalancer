package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// defaultTLSCertDir is the default directory for TLS certificates
	defaultTLSCertDir = "/etc/vpsie-lb/certs"
)

// TLSConfig represents TLS/SSL configuration
type TLSConfig struct {
	CertificatePath string   `json:"certificate_path" yaml:"certificate_path"`
	PrivateKeyPath  string   `json:"private_key_path" yaml:"private_key_path"`
	CACertPath      string   `json:"ca_cert_path,omitempty" yaml:"ca_cert_path,omitempty"`
	MinVersion      string   `json:"min_version" yaml:"min_version"` // TLSv1.2, TLSv1.3
	MaxVersion      string   `json:"max_version,omitempty" yaml:"max_version,omitempty"`
	CipherSuites    []string `json:"cipher_suites,omitempty" yaml:"cipher_suites,omitempty"`
	ALPN            []string `json:"alpn,omitempty" yaml:"alpn,omitempty"` // h2, http/1.1
}

// validateTLSFilePath validates that a TLS file path is within allowed directory
func validateTLSFilePath(path, allowedDir string) error {
	// Get absolute path
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Ensure allowed directory is also absolute
	absAllowedDir, err := filepath.Abs(allowedDir)
	if err != nil {
		return fmt.Errorf("invalid allowed directory: %w", err)
	}

	// Must be within allowed directory
	if !strings.HasPrefix(cleanPath, absAllowedDir+string(filepath.Separator)) &&
		cleanPath != absAllowedDir {
		return fmt.Errorf("path must be within %s", absAllowedDir)
	}

	// Resolve symlinks to prevent symlink escape
	evalPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to evaluate symlinks: %w", err)
	}
	if err == nil && evalPath != cleanPath {
		// Validate the resolved path is also within allowed directory
		if !strings.HasPrefix(evalPath, absAllowedDir+string(filepath.Separator)) &&
			evalPath != absAllowedDir {
			return fmt.Errorf("symlink points outside allowed directory: %s -> %s", cleanPath, evalPath)
		}
	}

	return nil
}

// Validate validates the TLS configuration
func (t *TLSConfig) Validate() error {
	if t.CertificatePath == "" {
		return ErrMissingCertificate
	}
	if t.PrivateKeyPath == "" {
		return ErrMissingPrivateKey
	}

	// Validate certificate path is within allowed directory
	if err := validateTLSFilePath(t.CertificatePath, defaultTLSCertDir); err != nil {
		return fmt.Errorf("invalid certificate path: %w", err)
	}

	// Validate private key path is within allowed directory
	if err := validateTLSFilePath(t.PrivateKeyPath, defaultTLSCertDir); err != nil {
		return fmt.Errorf("invalid private key path: %w", err)
	}

	// Validate CA cert path if provided
	if t.CACertPath != "" {
		if err := validateTLSFilePath(t.CACertPath, defaultTLSCertDir); err != nil {
			return fmt.Errorf("invalid CA certificate path: %w", err)
		}
	}

	// Validate TLS version
	validVersions := map[string]bool{
		"TLSv1.2": true,
		"TLSv1.3": true,
	}
	if !validVersions[t.MinVersion] {
		return ErrInvalidTLSVersion
	}
	if t.MaxVersion != "" && !validVersions[t.MaxVersion] {
		return ErrInvalidTLSVersion
	}

	return nil
}

// GetDefaultCipherSuites returns a secure default cipher suite list
func GetDefaultCipherSuites() []string {
	return []string{
		"ECDHE-ECDSA-AES128-GCM-SHA256",
		"ECDHE-RSA-AES128-GCM-SHA256",
		"ECDHE-ECDSA-AES256-GCM-SHA384",
		"ECDHE-RSA-AES256-GCM-SHA384",
		"ECDHE-ECDSA-CHACHA20-POLY1305",
		"ECDHE-RSA-CHACHA20-POLY1305",
	}
}

// GetDefaultALPN returns default ALPN protocols
func GetDefaultALPN() []string {
	return []string{"h2", "http/1.1"}
}
