package models

import (
	"reflect"
	"testing"
)

func TestTLSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tls     TLSConfig
		wantErr error
	}{
		{
			name: "valid TLS config with TLSv1.2",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
			},
			wantErr: nil,
		},
		{
			name: "valid TLS config with TLSv1.3",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.3",
			},
			wantErr: nil,
		},
		{
			name: "valid TLS config with min and max version",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
				MaxVersion:      "TLSv1.3",
			},
			wantErr: nil,
		},
		{
			name: "valid TLS config with cipher suites",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
				CipherSuites:    []string{"ECDHE-RSA-AES128-GCM-SHA256"},
			},
			wantErr: nil,
		},
		{
			name: "valid TLS config with ALPN",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
				ALPN:            []string{"h2", "http/1.1"},
			},
			wantErr: nil,
		},
		{
			name: "valid TLS config with CA cert",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				CACertPath:      "/etc/certs/ca.pem",
				MinVersion:      "TLSv1.2",
			},
			wantErr: nil,
		},
		{
			name: "missing certificate path",
			tls: TLSConfig{
				PrivateKeyPath: "/etc/certs/key.pem",
				MinVersion:     "TLSv1.2",
			},
			wantErr: ErrMissingCertificate,
		},
		{
			name: "missing private key path",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				MinVersion:      "TLSv1.2",
			},
			wantErr: ErrMissingPrivateKey,
		},
		{
			name: "invalid min version - TLSv1.1",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.1",
			},
			wantErr: ErrInvalidTLSVersion,
		},
		{
			name: "invalid min version - TLSv1.0",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.0",
			},
			wantErr: ErrInvalidTLSVersion,
		},
		{
			name: "invalid min version - empty",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "",
			},
			wantErr: ErrInvalidTLSVersion,
		},
		{
			name: "invalid max version",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
				MaxVersion:      "TLSv1.1",
			},
			wantErr: ErrInvalidTLSVersion,
		},
		{
			name: "invalid max version - arbitrary string",
			tls: TLSConfig{
				CertificatePath: "/etc/certs/cert.pem",
				PrivateKeyPath:  "/etc/certs/key.pem",
				MinVersion:      "TLSv1.2",
				MaxVersion:      "invalid",
			},
			wantErr: ErrInvalidTLSVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tls.Validate()
			if err != tt.wantErr {
				t.Errorf("TLSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultCipherSuites(t *testing.T) {
	suites := GetDefaultCipherSuites()

	if len(suites) == 0 {
		t.Error("GetDefaultCipherSuites() returned empty slice")
	}

	expectedSuites := []string{
		"ECDHE-ECDSA-AES128-GCM-SHA256",
		"ECDHE-RSA-AES128-GCM-SHA256",
		"ECDHE-ECDSA-AES256-GCM-SHA384",
		"ECDHE-RSA-AES256-GCM-SHA384",
		"ECDHE-ECDSA-CHACHA20-POLY1305",
		"ECDHE-RSA-CHACHA20-POLY1305",
	}

	if !reflect.DeepEqual(suites, expectedSuites) {
		t.Errorf("GetDefaultCipherSuites() = %v, want %v", suites, expectedSuites)
	}
}

func TestGetDefaultALPN(t *testing.T) {
	alpn := GetDefaultALPN()

	if len(alpn) == 0 {
		t.Error("GetDefaultALPN() returned empty slice")
	}

	expectedALPN := []string{"h2", "http/1.1"}

	if !reflect.DeepEqual(alpn, expectedALPN) {
		t.Errorf("GetDefaultALPN() = %v, want %v", alpn, expectedALPN)
	}
}
