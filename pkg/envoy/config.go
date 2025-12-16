package envoy

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigManager manages Envoy configuration files
type ConfigManager struct {
	validator *Validator
	configDir string
}

// NewConfigManager creates a new Envoy config manager
func NewConfigManager(configDir string, validator *Validator) *ConfigManager {
	return &ConfigManager{
		configDir: configDir,
		validator: validator,
	}
}

// WriteListeners writes the listeners configuration to file
func (cm *ConfigManager) WriteListeners(data []byte) error {
	return cm.writeConfigFile("listeners.yaml", data)
}

// WriteClusters writes the clusters configuration to file
func (cm *ConfigManager) WriteClusters(data []byte) error {
	return cm.writeConfigFile("clusters.yaml", data)
}

// WriteBootstrap writes the bootstrap configuration to file
func (cm *ConfigManager) WriteBootstrap(data []byte) error {
	bootstrapPath := filepath.Join(filepath.Dir(cm.configDir), "bootstrap.yaml")
	return cm.atomicWrite(bootstrapPath, data)
}

// ApplyConfig applies a complete Envoy configuration
func (cm *ConfigManager) ApplyConfig(config *EnvoyConfig) error {
	// Write listeners
	if err := cm.WriteListeners(config.Listeners); err != nil {
		return fmt.Errorf("failed to write listeners: %w", err)
	}

	// Write clusters
	if err := cm.WriteClusters(config.Clusters); err != nil {
		return fmt.Errorf("failed to write clusters: %w", err)
	}

	return nil
}

// BackupConfig backs up the current configuration
func (cm *ConfigManager) BackupConfig() error {
	backupDir := filepath.Join(cm.configDir, ".backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	files := []string{"listeners.yaml", "clusters.yaml"}
	for _, file := range files {
		src := filepath.Join(cm.configDir, file)
		dst := filepath.Join(backupDir, file)

		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip if file doesn't exist
			}
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if err = os.WriteFile(dst, data, 0600); err != nil {
			return fmt.Errorf("failed to backup %s: %w", file, err)
		}
	}

	return nil
}

// RestoreConfig restores the configuration from backup
func (cm *ConfigManager) RestoreConfig() error {
	backupDir := filepath.Join(cm.configDir, ".backup")

	files := []string{"listeners.yaml", "clusters.yaml"}
	for _, file := range files {
		src := filepath.Join(backupDir, file)
		dst := filepath.Join(cm.configDir, file)

		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip if backup doesn't exist
			}
			return fmt.Errorf("failed to read backup %s: %w", file, err)
		}

		// #nosec G306 -- Config files need 0644 to allow Envoy process (different user) to read them
		if err = os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("failed to restore %s: %w", file, err)
		}
	}

	return nil
}

// writeConfigFile writes a configuration file atomically
func (cm *ConfigManager) writeConfigFile(filename string, data []byte) error {
	path := filepath.Join(cm.configDir, filename)
	return cm.atomicWrite(path, data)
}

// atomicWrite writes data to a file atomically using a temp file
func (cm *ConfigManager) atomicWrite(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file
	tmpPath := path + ".tmp"
	// #nosec G306 -- Config files need 0644 to allow Envoy process (different user) to read them
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Cleanup on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
