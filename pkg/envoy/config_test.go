package envoy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigManager(t *testing.T) {
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager("/etc/envoy", validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cm.configDir != "/etc/envoy" {
		t.Errorf("configDir = %v, want /etc/envoy", cm.configDir)
	}
	if cm.validator != validator {
		t.Error("validator not set correctly")
	}
}

func TestConfigManager_WriteListeners(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data := []byte("listeners:\n  - name: test\n")
	err = cm.WriteListeners(data)

	if err != nil {
		t.Errorf("WriteListeners() error = %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "listeners.yaml"))
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %v, want %v", string(content), string(data))
	}
}

func TestConfigManager_WriteClusters(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data := []byte("clusters:\n  - name: test\n")
	err = cm.WriteClusters(data)

	if err != nil {
		t.Errorf("WriteClusters() error = %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "clusters.yaml"))
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %v, want %v", string(content), string(data))
	}
}

func TestConfigManager_WriteBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	os.MkdirAll(configDir, 0755)

	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(configDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data := []byte("node:\n  id: test\n")
	err = cm.WriteBootstrap(data)

	if err != nil {
		t.Errorf("WriteBootstrap() error = %v", err)
	}

	// Verify file was written to parent directory
	bootstrapPath := filepath.Join(tmpDir, "bootstrap.yaml")
	content, err := os.ReadFile(bootstrapPath)
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %v, want %v", string(content), string(data))
	}
}

func TestConfigManager_ApplyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	config := &EnvoyConfig{
		Listeners: []byte("listeners:\n  - name: test\n"),
		Clusters:  []byte("clusters:\n  - name: test\n"),
	}

	err = cm.ApplyConfig(config)
	if err != nil {
		t.Errorf("ApplyConfig() error = %v", err)
	}

	// Verify both files were written
	listenersPath := filepath.Join(tmpDir, "listeners.yaml")
	if _, statErr := os.Stat(listenersPath); os.IsNotExist(statErr) {
		t.Error("listeners.yaml was not created")
	}

	clustersPath := filepath.Join(tmpDir, "clusters.yaml")
	if _, statErr := os.Stat(clustersPath); os.IsNotExist(statErr) {
		t.Error("clusters.yaml was not created")
	}
}

func TestConfigManager_BackupConfig(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Create initial config files
	listenersData := []byte("listeners:\n  - name: test\n")
	clustersData := []byte("clusters:\n  - name: test\n")

	os.WriteFile(filepath.Join(tmpDir, "listeners.yaml"), listenersData, 0600)
	os.WriteFile(filepath.Join(tmpDir, "clusters.yaml"), clustersData, 0600)

	// Backup
	err = cm.BackupConfig()
	if err != nil {
		t.Errorf("BackupConfig() error = %v", err)
	}

	// Verify backup files exist
	backupDir := filepath.Join(tmpDir, ".backup")
	backupListeners := filepath.Join(backupDir, "listeners.yaml")
	backupClusters := filepath.Join(backupDir, "clusters.yaml")

	if _, statErr := os.Stat(backupListeners); os.IsNotExist(statErr) {
		t.Error("Backup listeners.yaml was not created")
	}
	if _, statErr := os.Stat(backupClusters); os.IsNotExist(statErr) {
		t.Error("Backup clusters.yaml was not created")
	}

	// Verify backup content
	content, _ := os.ReadFile(backupListeners)
	if string(content) != string(listenersData) {
		t.Error("Backup listeners content doesn't match")
	}
}

func TestConfigManager_BackupConfig_MissingFiles(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Backup with no files should not error
	err = cm.BackupConfig()
	if err != nil {
		t.Errorf("BackupConfig() should not error on missing files, got: %v", err)
	}
}

func TestConfigManager_RestoreConfig(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Create backup files
	backupDir := filepath.Join(tmpDir, ".backup")
	os.MkdirAll(backupDir, 0755)

	listenersData := []byte("listeners:\n  - name: backup\n")
	clustersData := []byte("clusters:\n  - name: backup\n")

	os.WriteFile(filepath.Join(backupDir, "listeners.yaml"), listenersData, 0600)
	os.WriteFile(filepath.Join(backupDir, "clusters.yaml"), clustersData, 0600)

	// Create different current files
	os.WriteFile(filepath.Join(tmpDir, "listeners.yaml"), []byte("different"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "clusters.yaml"), []byte("different"), 0600)

	// Restore
	err = cm.RestoreConfig()
	if err != nil {
		t.Errorf("RestoreConfig() error = %v", err)
	}

	// Verify files were restored
	content, _ := os.ReadFile(filepath.Join(tmpDir, "listeners.yaml"))
	if string(content) != string(listenersData) {
		t.Error("listeners.yaml was not restored correctly")
	}

	content, _ = os.ReadFile(filepath.Join(tmpDir, "clusters.yaml"))
	if string(content) != string(clustersData) {
		t.Error("clusters.yaml was not restored correctly")
	}
}

func TestConfigManager_RestoreConfig_NoBackup(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Restore with no backup should not error
	err = cm.RestoreConfig()
	if err != nil {
		t.Errorf("RestoreConfig() should not error when no backup exists, got: %v", err)
	}
}

func TestConfigManager_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	testPath := filepath.Join(tmpDir, "test.yaml")
	data := []byte("test data")

	err = cm.atomicWrite(testPath, data)
	if err != nil {
		t.Errorf("atomicWrite() error = %v", err)
	}

	// Verify file was written
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Errorf("Failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %v, want %v", string(content), string(data))
	}

	// Verify temp file was removed
	tmpPath := testPath + ".tmp"
	if _, statErr := os.Stat(tmpPath); !os.IsNotExist(statErr) {
		t.Error("Temp file was not cleaned up")
	}
}

func TestConfigManager_AtomicWrite_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewValidator("/usr/bin/envoy")
	cm, err := NewConfigManager(tmpDir, validator)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Write to a subdirectory that doesn't exist
	testPath := filepath.Join(tmpDir, "subdir", "test.yaml")
	data := []byte("test data")

	err = cm.atomicWrite(testPath, data)
	if err != nil {
		t.Errorf("atomicWrite() error = %v", err)
	}

	// Verify directory was created
	if _, statErr := os.Stat(filepath.Dir(testPath)); os.IsNotExist(statErr) {
		t.Error("Directory was not created")
	}

	// Verify file was written
	content, readErr := os.ReadFile(testPath)
	if readErr != nil {
		t.Errorf("Failed to read written file: %v", readErr)
	}
	if string(content) != string(data) {
		t.Errorf("File content = %v, want %v", string(content), string(data))
	}
}
