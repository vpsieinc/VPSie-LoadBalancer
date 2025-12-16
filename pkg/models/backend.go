package models

// Backend represents a backend server
type Backend struct {
	ID      string `json:"id" yaml:"id"`
	Address string `json:"address" yaml:"address"`                   // IP or hostname
	Status  string `json:"status,omitempty" yaml:"status,omitempty"` // up, down, unknown
	Port    int    `json:"port" yaml:"port"`
	Weight  int    `json:"weight,omitempty" yaml:"weight,omitempty"`
	Enabled bool   `json:"enabled" yaml:"enabled"`
}

// Validate validates the backend configuration
func (b *Backend) Validate() error {
	if b.ID == "" {
		return ErrInvalidBackendID
	}
	if b.Address == "" {
		return ErrInvalidBackendAddress
	}
	if b.Port <= 0 || b.Port > 65535 {
		return ErrInvalidBackendPort
	}
	if b.Weight < 0 {
		return ErrInvalidBackendWeight
	}
	return nil
}

// IsHealthy returns true if the backend is in healthy state
func (b *Backend) IsHealthy() bool {
	return b.Enabled && b.Status == "up"
}
