package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name: "valid config with all fields",
			configYAML: `
vpsie:
  api_url: "https://api.vpsie.com/v1"
  api_key_file: "/etc/vpsie/api-key"
  loadbalancer_id: "lb-12345"
  poll_interval: 60s
envoy:
  config_path: "/etc/envoy"
  admin_address: "127.0.0.1:9901"
  binary_path: "/usr/local/bin/envoy"
logging:
  level: "debug"
  format: "json"
`,
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if c.VPSie.APIURL != "https://api.vpsie.com/v1" {
					t.Errorf("APIURL = %v, want https://api.vpsie.com/v1", c.VPSie.APIURL)
				}
				if c.VPSie.LoadBalancerID != "lb-12345" {
					t.Errorf("LoadBalancerID = %v, want lb-12345", c.VPSie.LoadBalancerID)
				}
				if c.VPSie.PollInterval != 60*time.Second {
					t.Errorf("PollInterval = %v, want 60s", c.VPSie.PollInterval)
				}
				if c.Envoy.ConfigPath != "/etc/envoy" {
					t.Errorf("ConfigPath = %v, want /etc/envoy", c.Envoy.ConfigPath)
				}
				if c.Logging.Level != "debug" {
					t.Errorf("Logging Level = %v, want debug", c.Logging.Level)
				}
			},
		},
		{
			name: "config with defaults",
			configYAML: `
vpsie:
  api_url: "https://api.vpsie.com/v1"
  api_key_file: "/etc/vpsie/api-key"
  loadbalancer_id: "lb-12345"
envoy:
  config_path: "/etc/envoy"
`,
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if c.VPSie.PollInterval != 30*time.Second {
					t.Errorf("PollInterval = %v, want default 30s", c.VPSie.PollInterval)
				}
				if c.Envoy.AdminAddress != "127.0.0.1:9901" {
					t.Errorf("AdminAddress = %v, want default 127.0.0.1:9901", c.Envoy.AdminAddress)
				}
				if c.Envoy.BinaryPath != "/usr/bin/envoy" {
					t.Errorf("BinaryPath = %v, want default /usr/bin/envoy", c.Envoy.BinaryPath)
				}
				if c.Logging.Level != "info" {
					t.Errorf("Logging Level = %v, want default info", c.Logging.Level)
				}
				if c.Logging.Format != "json" {
					t.Errorf("Logging Format = %v, want default json", c.Logging.Format)
				}
			},
		},
		{
			name:       "invalid YAML",
			configYAML: `invalid: [yaml: content`,
			wantErr:    true,
		},
		{
			name: "empty config",
			configYAML: `
vpsie: {}
envoy: {}
`,
			wantErr: false,
			validate: func(t *testing.T, c *Config) {
				if c.VPSie.PollInterval != 30*time.Second {
					t.Errorf("Expected default poll interval")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0600)
			if err != nil {
				t.Fatalf("Failed to write temp config: %v", err)
			}

			// Load the config
			config, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run validation if provided
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, config)
			}
		})
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

func TestVPSieConfig_LoadAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		keyContent string
		wantErr    bool
		expected   string
	}{
		{
			name:       "valid API key",
			keyContent: "my-secret-api-key-123",
			wantErr:    false,
			expected:   "my-secret-api-key-123",
		},
		{
			name:       "API key with whitespace",
			keyContent: "  my-api-key  \n",
			wantErr:    false,
			expected:   "my-api-key",
		},
		{
			name:       "API key with newlines",
			keyContent: "\n\nmy-api-key\n\n",
			wantErr:    false,
			expected:   "my-api-key",
		},
		{
			name:       "empty file",
			keyContent: "",
			wantErr:    true,
		},
		{
			name:       "only whitespace",
			keyContent: "   \n\n  ",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary API key file
			tmpDir := t.TempDir()
			keyPath := filepath.Join(tmpDir, "api-key")
			err := os.WriteFile(keyPath, []byte(tt.keyContent), 0600)
			if err != nil {
				t.Fatalf("Failed to write temp key file: %v", err)
			}

			// Load the API key
			cfg := VPSieConfig{APIKeyFile: keyPath}
			apiKey, err := cfg.LoadAPIKey()

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAPIKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && apiKey != tt.expected {
				t.Errorf("LoadAPIKey() = %v, want %v", apiKey, tt.expected)
			}
		})
	}
}

func TestVPSieConfig_LoadAPIKey_FileNotFound(t *testing.T) {
	cfg := VPSieConfig{APIKeyFile: "/nonexistent/api-key"}
	_, err := cfg.LoadAPIKey()
	if err == nil {
		t.Error("Expected error when loading non-existent API key file")
	}
}
