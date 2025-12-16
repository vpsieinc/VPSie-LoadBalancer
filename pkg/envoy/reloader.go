package envoy

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

// Reloader handles hot reloading of Envoy configuration
type Reloader struct {
	envoyBinary  string
	configPath   string
	pidFile      string
	currentEpoch atomic.Int32
	mu           sync.Mutex // Protects Reload() from concurrent execution
}

// NewReloader creates a new Envoy reloader
func NewReloader(envoyBinary, configPath, pidFile string) *Reloader {
	return &Reloader{
		envoyBinary: envoyBinary,
		configPath:  configPath,
		pidFile:     pidFile,
		// currentEpoch defaults to 0 (zero value of atomic.Int32)
	}
}

// Reload performs a hot restart of Envoy with the new configuration
func (r *Reloader) Reload() error {
	// Ensure only one reload happens at a time to prevent epoch desynchronization
	r.mu.Lock()
	defer r.mu.Unlock()

	// Increment epoch atomically
	newEpoch := r.currentEpoch.Add(1)

	// Build command for hot restart
	// #nosec G204 -- envoyBinary is set at initialization, not from user input
	cmd := exec.Command(
		r.envoyBinary,
		"-c", r.configPath,
		"--restart-epoch", strconv.Itoa(int(newEpoch)),
		"--parent-shutdown-time-s", "10",
	)

	// Start the new Envoy process (detached, will continue running)
	if err := cmd.Start(); err != nil {
		r.currentEpoch.Add(-1) // Rollback epoch on failure
		return fmt.Errorf("failed to start new Envoy process: %w", err)
	}

	// Release the process handle - Envoy will continue running independently
	// The hot restart mechanism will handle the transition between old and new processes
	//nolint:errcheck // Intentionally ignore - process will continue running even if release fails
	cmd.Process.Release()

	return nil
}

// ReloadGraceful sends SIGHUP to the running Envoy process for graceful reload
func (r *Reloader) ReloadGraceful() error {
	// Read PID from file
	pidData, err := os.ReadFile(r.pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	// Trim whitespace and newlines to prevent injection attacks
	pidStr := strings.TrimSpace(string(pidData))

	// Validate PID format (must be positive integer)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	// Validate PID range (must be positive and within reasonable bounds)
	// Linux max PID is typically 4194304, Darwin/macOS max is 99999
	const maxPID = 4194304
	if pid <= 0 || pid > maxPID {
		return fmt.Errorf("PID out of valid range: %d (must be between 1 and %d)", pid, maxPID)
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find Envoy process: %w", err)
	}

	// Send SIGHUP signal
	if err = process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to Envoy: %w", err)
	}

	return nil
}

// GetCurrentEpoch returns the current restart epoch
func (r *Reloader) GetCurrentEpoch() int {
	return int(r.currentEpoch.Load())
}
