package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/vpsie/vpsie-loadbalancer/pkg/envoy"
	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
)

// Agent is the main control plane agent
type Agent struct {
	config         *Config
	vpsieClient    *VPSieClient
	envoyGenerator *envoy.Generator
	envoyManager   *envoy.ConfigManager
	envoyValidator *envoy.Validator
	envoyReloader  *envoy.Reloader
	lastConfigHash string
	running        bool
}

// NewAgent creates a new agent instance
func NewAgent(cfg *Config) (*Agent, error) {
	// Load API key
	apiKey, err := cfg.VPSie.LoadAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load API key: %w", err)
	}

	// Create VPSie client
	vpsieClient := NewVPSieClient(
		apiKey,
		cfg.VPSie.APIURL,
		cfg.VPSie.LoadBalancerID,
	)

	// Create Envoy components
	envoyGenerator := envoy.NewGenerator(
		cfg.VPSie.LoadBalancerID,
		cfg.Envoy.ConfigPath,
		cfg.Envoy.AdminAddress,
		9901,  // default admin port
		50000, // max connections
	)

	envoyValidator := envoy.NewValidator(cfg.Envoy.BinaryPath)
	envoyManager := envoy.NewConfigManager(cfg.Envoy.ConfigPath, envoyValidator)
	envoyReloader := envoy.NewReloader(
		cfg.Envoy.BinaryPath,
		cfg.Envoy.ConfigPath+"/bootstrap.yaml",
		"/var/run/envoy.pid",
	)

	return &Agent{
		config:         cfg,
		vpsieClient:    vpsieClient,
		envoyGenerator: envoyGenerator,
		envoyManager:   envoyManager,
		envoyValidator: envoyValidator,
		envoyReloader:  envoyReloader,
		running:        false,
	}, nil
}

// Start starts the agent's reconciliation loop
func (a *Agent) Start(ctx context.Context) error {
	log.Printf("Starting VPSie Load Balancer Agent...")
	log.Printf("Load Balancer ID: %s", a.config.VPSie.LoadBalancerID)
	log.Printf("Poll Interval: %s", a.config.VPSie.PollInterval)

	a.running = true

	// Initial sync
	if err := a.syncConfiguration(); err != nil {
		log.Printf("Warning: Initial configuration sync failed: %v", err)
		// Don't fail on initial sync error, continue and retry
	}

	// Start reconciliation loop
	ticker := time.NewTicker(a.config.VPSie.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopping...")
			a.running = false
			return nil

		case <-ticker.C:
			if err := a.syncConfiguration(); err != nil {
				log.Printf("Error syncing configuration: %v", err)
			}
		}
	}
}

// syncConfiguration fetches config from VPSie and applies it to Envoy
func (a *Agent) syncConfiguration() error {
	log.Println("Syncing configuration from VPSie API...")

	// Fetch current configuration
	lb, err := a.vpsieClient.GetLoadBalancerConfig()
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}

	// Validate configuration
	if err = lb.Validate(); err != nil {
		return fmt.Errorf("invalid configuration from VPSie: %w", err)
	}

	// Check if configuration has changed
	configHash := a.computeConfigHash(lb)
	if configHash == a.lastConfigHash {
		log.Println("Configuration unchanged, skipping update")
		return nil
	}

	log.Printf("Configuration changed, applying new config (hash: %s)", configHash)

	// Backup current configuration
	if err = a.envoyManager.BackupConfig(); err != nil {
		log.Printf("Warning: Failed to backup config: %v", err)
	}

	// Generate new Envoy configuration
	var envoyConfig *envoy.EnvoyConfig
	envoyConfig, err = a.envoyGenerator.GenerateFullConfig(lb)
	if err != nil {
		return fmt.Errorf("failed to generate Envoy config: %w", err)
	}

	// Apply configuration
	if err = a.envoyManager.ApplyConfig(envoyConfig); err != nil {
		return fmt.Errorf("failed to apply config: %w", err)
	}

	// Reload Envoy (hot restart)
	log.Println("Reloading Envoy with new configuration...")
	if err = a.reloadEnvoy(); err != nil {
		// Restore backup on failure
		log.Printf("Reload failed, restoring backup: %v", err)
		if restoreErr := a.envoyManager.RestoreConfig(); restoreErr != nil {
			log.Printf("Failed to restore backup: %v", restoreErr)
		}
		return fmt.Errorf("failed to reload Envoy: %w", err)
	}

	// Update last config hash
	a.lastConfigHash = configHash

	// Notify VPSie of successful update
	if err = a.vpsieClient.SendEvent("config_updated", "Configuration successfully updated", map[string]interface{}{
		"config_hash": configHash,
		"epoch":       a.envoyReloader.GetCurrentEpoch(),
	}); err != nil {
		log.Printf("Warning: Failed to send update event: %v", err)
	}

	log.Println("Configuration sync completed successfully")
	return nil
}

// reloadEnvoy performs a hot reload of Envoy
func (a *Agent) reloadEnvoy() error {
	// Note: In a real implementation, you might want to use graceful reload
	// For now, we'll assume Envoy is managed by systemd and we just need
	// to write the config files. Envoy will detect changes and reload.

	// In production, you could:
	// 1. Use Envoy's hot restart mechanism
	// 2. Signal systemd to reload the service
	// 3. Use Envoy's xDS API for dynamic config updates

	log.Println("Envoy configuration files updated, Envoy will detect changes")
	return nil
}

// computeConfigHash computes a cryptographic hash of the configuration for change detection
func (a *Agent) computeConfigHash(lb *models.LoadBalancer) string {
	// Marshal the entire configuration to JSON to capture all changes
	data, err := json.Marshal(lb)
	if err != nil {
		// Fallback to a timestamp-based hash if marshaling fails
		log.Printf("Warning: Failed to marshal config for hashing: %v", err)
		return fmt.Sprintf("%s-%d-%d", lb.UpdatedAt.Format(time.RFC3339), len(lb.Backends), lb.Port)
	}

	// Compute SHA-256 hash of the JSON representation
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// IsRunning returns true if the agent is running
func (a *Agent) IsRunning() bool {
	return a.running
}

// Stop stops the agent
func (a *Agent) Stop() {
	log.Println("Stopping agent...")
	a.running = false
}
