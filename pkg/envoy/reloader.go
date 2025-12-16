package envoy

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
	"syscall"
)

// Reloader handles hot reloading of Envoy configuration
type Reloader struct {
	envoyBinary  string
	configPath   string
	pidFile      string
	currentEpoch atomic.Int32
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
	if err := cmd.Process.Release(); err != nil {
		// Process started successfully but we couldn't release the handle
		// This is not critical - log but don't fail
		// The process will still continue running
	}

	return nil
}

// ReloadGraceful sends SIGHUP to the running Envoy process for graceful reload
func (r *Reloader) ReloadGraceful() error {
	// Read PID from file
	pidData, err := os.ReadFile(r.pidFile)
	if err != nil {
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return fmt.Errorf("invalid PID in file: %w", err)
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
