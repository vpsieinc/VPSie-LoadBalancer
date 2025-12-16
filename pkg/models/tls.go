package models

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

// Validate validates the TLS configuration
func (t *TLSConfig) Validate() error {
	if t.CertificatePath == "" {
		return ErrMissingCertificate
	}
	if t.PrivateKeyPath == "" {
		return ErrMissingPrivateKey
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
