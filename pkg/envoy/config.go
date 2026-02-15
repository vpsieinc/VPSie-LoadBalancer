package envoy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigManager manages Envoy configuration files
type ConfigManager struct {
	validator *Validator
	configDir string
	baseDir   string // Parent of configDir for bootstrap file
}

// NewConfigManager creates a new Envoy config manager
func NewConfigManager(configDir string, validator *Validator) (*ConfigManager, error) {
	// Validate and sanitize config directory path
	cleanConfigDir, err := filepath.Abs(filepath.Clean(configDir))
	if err != nil {
		return nil, fmt.Errorf("invalid config directory: %w", err)
	}

	// Store parent directory for bootstrap file validation
	baseDir := filepath.Dir(cleanConfigDir)

	return &ConfigManager{
		configDir: cleanConfigDir,
		baseDir:   baseDir,
		validator: validator,
	}, nil
}

// validatePath ensures the given path is within allowed directories
func (cm *ConfigManager) validatePath(path string) error {
	// Clean and get absolute path
	cleanPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path is within configDir OR baseDir (for bootstrap)
	inConfigDir := cleanPath == cm.configDir ||
		strings.HasPrefix(cleanPath, cm.configDir+string(filepath.Separator))
	inBaseDir := cleanPath == cm.baseDir ||
		strings.HasPrefix(cleanPath, cm.baseDir+string(filepath.Separator))

	if !inConfigDir && !inBaseDir {
		return fmt.Errorf("path traversal attempt detected: %s not within allowed directories", cleanPath)
	}

	// Additional check: ensure no symlinks point outside allowed directories
	evalPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to evaluate symlinks: %w", err)
	}
	if err == nil && evalPath != cleanPath {
		// Validate the resolved path too
		inConfigDirResolved := evalPath == cm.configDir ||
			strings.HasPrefix(evalPath, cm.configDir+string(filepath.Separator))
		inBaseDirResolved := evalPath == cm.baseDir ||
			strings.HasPrefix(evalPath, cm.baseDir+string(filepath.Separator))

		if !inConfigDirResolved && !inBaseDirResolved {
			return fmt.Errorf("symlink points outside allowed directories: %s -> %s", cleanPath, evalPath)
		}
	}

	return nil
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
	// Validate path to prevent traversal attacks
	if err := cm.validatePath(path); err != nil {
		return err
	}

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
		_ = os.Remove(tmpPath) // Cleanup on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
