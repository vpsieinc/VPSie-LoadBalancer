package envoy

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// Reloader handles hot reloading of Envoy configuration
type Reloader struct {
	envoyBinary  string
	configPath   string
	pidFile      string
	currentEpoch int
}

// NewReloader creates a new Envoy reloader
func NewReloader(envoyBinary, configPath, pidFile string) *Reloader {
	return &Reloader{
		envoyBinary:  envoyBinary,
		configPath:   configPath,
		pidFile:      pidFile,
		currentEpoch: 0,
	}
}

// Reload performs a hot restart of Envoy with the new configuration
func (r *Reloader) Reload() error {
	// Increment epoch
	r.currentEpoch++

	// Build command for hot restart
	// #nosec G204 -- envoyBinary is set at initialization, not from user input
	cmd := exec.Command(
		r.envoyBinary,
		"-c", r.configPath,
		"--restart-epoch", strconv.Itoa(r.currentEpoch),
		"--parent-shutdown-time-s", "10",
	)

	// Start the new Envoy process
	if err := cmd.Start(); err != nil {
		r.currentEpoch-- // Rollback epoch on failure
		return fmt.Errorf("failed to start new Envoy process: %w", err)
	}

	// Wait for the process to complete initialization
	if err := cmd.Wait(); err != nil {
		// Check if it's a normal hot restart exit
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 0 means success
			if exitErr.ExitCode() != 0 {
				r.currentEpoch-- // Rollback epoch on failure
				return fmt.Errorf("Envoy process exited with error: %w", err)
			}
		}
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
	if signalErr := process.Signal(syscall.SIGHUP); signalErr != nil {
		return fmt.Errorf("failed to send SIGHUP to Envoy: %w", signalErr)
	}

	return nil
}

// GetCurrentEpoch returns the current restart epoch
func (r *Reloader) GetCurrentEpoch() int {
	return r.currentEpoch
}
