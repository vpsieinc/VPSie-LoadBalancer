package agent

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the agent configuration
type Config struct {
	Envoy   EnvoySettings `yaml:"envoy"`
	Logging LoggingConfig `yaml:"logging"`
	VPSie   VPSieConfig   `yaml:"vpsie"`
}

// VPSieConfig contains VPSie API configuration
type VPSieConfig struct {
	APIURL         string        `yaml:"api_url"`
	APIKeyFile     string        `yaml:"api_key_file"`
	LoadBalancerID string        `yaml:"loadbalancer_id"`
	PollInterval   time.Duration `yaml:"poll_interval"`
}

// EnvoySettings contains Envoy-specific configuration
type EnvoySettings struct {
	ConfigPath     string `yaml:"config_path"`
	AdminAddress   string `yaml:"admin_address"`
	BinaryPath     string `yaml:"binary_path"`
	PidFile        string `yaml:"pid_file"`
	AdminPort      int    `yaml:"admin_port"`
	MaxConnections int    `yaml:"max_connections"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// LoadConfig loads the agent configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.VPSie.PollInterval == 0 {
		config.VPSie.PollInterval = 30 * time.Second
	}
	if config.Envoy.AdminAddress == "" {
		config.Envoy.AdminAddress = "127.0.0.1:9901"
	}
	if config.Envoy.AdminPort == 0 {
		config.Envoy.AdminPort = 9901
	}
	if config.Envoy.MaxConnections == 0 {
		config.Envoy.MaxConnections = 50000
	}
	if config.Envoy.PidFile == "" {
		config.Envoy.PidFile = "/var/run/envoy.pid"
	}
	if config.Envoy.BinaryPath == "" {
		config.Envoy.BinaryPath = "/usr/bin/envoy"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	return &config, nil
}

// LoadAPIKey reads the API key from the configured file
func (c *VPSieConfig) LoadAPIKey() (string, error) {
	data, err := os.ReadFile(c.APIKeyFile)
	if err != nil {
		return "", fmt.Errorf("failed to read API key file: %w", err)
	}

	// Trim whitespace and newlines
	apiKey := string(data)
	apiKey = string(bytes.TrimSpace([]byte(apiKey)))

	if apiKey == "" {
		return "", fmt.Errorf("API key file is empty")
	}

	return apiKey, nil
}
