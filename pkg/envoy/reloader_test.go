package envoy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReloader_EpochIncrement(t *testing.T) {
	r := NewReloader("/nonexistent/envoy", "/tmp/envoy.yaml", "/tmp/envoy.pid")

	if r.GetCurrentEpoch() != 0 {
		t.Fatalf("expected initial epoch 0, got %d", r.GetCurrentEpoch())
	}

	// Reload will fail (binary doesn't exist) but epoch should still increment
	reloadErr := r.Reload()
	if reloadErr == nil {
		t.Fatal("expected error from Reload with nonexistent binary")
	}

	if r.GetCurrentEpoch() != 1 {
		t.Fatalf("expected epoch 1 after failed reload, got %d", r.GetCurrentEpoch())
	}

	reloadErr2 := r.Reload()
	if reloadErr2 == nil {
		t.Fatal("expected error from second Reload")
	}

	if r.GetCurrentEpoch() != 2 {
		t.Fatalf("expected epoch 2 after second reload, got %d", r.GetCurrentEpoch())
	}
}

func TestReloader_ReloadGraceful_MissingPIDFile(t *testing.T) {
	r := NewReloader("/usr/bin/envoy", "/tmp/envoy.yaml", "/nonexistent/envoy.pid")

	gracefulErr := r.ReloadGraceful()
	if gracefulErr == nil {
		t.Fatal("expected error when PID file is missing")
	}
}

func TestReloader_ReloadGraceful_NonNumericPID(t *testing.T) {
	// Create a temp PID file with non-numeric content
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "envoy.pid")

	writeErr := os.WriteFile(pidFile, []byte("not-a-number\n"), 0600)
	if writeErr != nil {
		t.Fatalf("failed to write PID file: %v", writeErr)
	}

	r := NewReloader("/usr/bin/envoy", "/tmp/envoy.yaml", pidFile)

	gracefulErr := r.ReloadGraceful()
	if gracefulErr == nil {
		t.Fatal("expected error for non-numeric PID")
	}
}

func TestReloader_ReloadGraceful_InvalidPIDRange(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "envoy.pid")

	writeErr := os.WriteFile(pidFile, []byte("-1\n"), 0600)
	if writeErr != nil {
		t.Fatalf("failed to write PID file: %v", writeErr)
	}

	r := NewReloader("/usr/bin/envoy", "/tmp/envoy.yaml", pidFile)

	gracefulErr := r.ReloadGraceful()
	if gracefulErr == nil {
		t.Fatal("expected error for negative PID")
	}
}
