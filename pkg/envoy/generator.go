package envoy

import (
	"bytes"
	_ "embed"
	"fmt"
	"net"
	"regexp"
	"text/template"

	"github.com/vpsie/vpsie-loadbalancer/pkg/models"
	"gopkg.in/yaml.v3"
)

var healthCheckPathRegex = regexp.MustCompile(`^/[a-zA-Z0-9/_\-.]*$`)

// validateHealthCheckPath validates that a health check path is safe for template rendering
func validateHealthCheckPath(path string) error {
	if path == "" {
		return nil
	}
	if !healthCheckPathRegex.MatchString(path) {
		return fmt.Errorf("invalid health check path %q: must start with / and contain only [a-zA-Z0-9/_\\-.]", path)
	}
	return nil
}

// validateAddress validates that an address is a valid hostname or IP, safe for template rendering
func validateAddress(addr string) error {
	if addr == "" {
		return fmt.Errorf("address must not be empty")
	}
	// Check if it's a valid IP
	if net.ParseIP(addr) != nil {
		return nil
	}
	// Check if it's a valid hostname
	if len(addr) > 253 {
		return fmt.Errorf("address %q too long", addr)
	}
	if !models.HostnameRegex.MatchString(addr) {
		return fmt.Errorf("invalid address %q: must be a valid hostname or IP", addr)
	}
	return nil
}

//go:embed templates/listener_http.yaml.tmpl
var listenerHTTPTemplate string

//go:embed templates/listener_https.yaml.tmpl
var listenerHTTPSTemplate string

//go:embed templates/listener_tcp.yaml.tmpl
var listenerTCPTemplate string

//go:embed templates/cluster.yaml.tmpl
var clusterTemplate string

//go:embed templates/bootstrap.yaml.tmpl
var bootstrapTemplate string

// Generator generates Envoy configuration from load balancer models
type Generator struct {
	nodeID         string
	configPath     string
	adminAddress   string
	adminPort      int
	maxConnections int
}

// NewGenerator creates a new Envoy config generator
func NewGenerator(nodeID, configPath, adminAddress string, adminPort, maxConnections int) *Generator {
	return &Generator{
		nodeID:         nodeID,
		configPath:     configPath,
		adminAddress:   adminAddress,
		adminPort:      adminPort,
		maxConnections: maxConnections,
	}
}

// GenerateBootstrap generates the Envoy bootstrap configuration
func (g *Generator) GenerateBootstrap() ([]byte, error) {
	tmpl, err := template.New("bootstrap").Parse(bootstrapTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bootstrap template: %w", err)
	}

	data := map[string]interface{}{
		"NodeID":         g.nodeID,
		"ConfigPath":     g.configPath,
		"AdminAddress":   g.adminAddress,
		"AdminPort":      g.adminPort,
		"MaxConnections": g.maxConnections,
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute bootstrap template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateListener generates an Envoy listener configuration
func (g *Generator) GenerateListener(lb *models.LoadBalancer) ([]byte, error) {
	var tmpl *template.Template
	var err error

	// Select template based on protocol
	switch lb.Protocol {
	case models.ProtocolHTTP:
		tmpl, err = template.New("listener").Parse(listenerHTTPTemplate)
	case models.ProtocolHTTPS:
		tmpl, err = template.New("listener").Parse(listenerHTTPSTemplate)
	case models.ProtocolTCP:
		tmpl, err = template.New("listener").Parse(listenerTCPTemplate)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", lb.Protocol)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse listener template: %w", err)
	}

	// Prepare template data
	data := map[string]interface{}{
		"Name":        fmt.Sprintf("listener_%s_%d", lb.Protocol, lb.Port),
		"Port":        lb.Port,
		"StatPrefix":  fmt.Sprintf("%s_%d", lb.Protocol, lb.Port),
		"ClusterName": fmt.Sprintf("cluster_%s", lb.ID),
	}

	// Add route config for HTTP/HTTPS
	if lb.Protocol == models.ProtocolHTTP || lb.Protocol == models.ProtocolHTTPS {
		data["RouteConfig"] = map[string]string{
			"Name":        "local_route",
			"VirtualHost": "backend",
		}
	}

	// Add TLS config for HTTPS
	if lb.Protocol == models.ProtocolHTTPS && lb.TLSConfig != nil {
		tlsData := map[string]interface{}{
			"CertificatePath": lb.TLSConfig.CertificatePath,
			"PrivateKeyPath":  lb.TLSConfig.PrivateKeyPath,
			"MinVersion":      lb.TLSConfig.MinVersion,
		}

		if lb.TLSConfig.MaxVersion != "" {
			tlsData["MaxVersion"] = lb.TLSConfig.MaxVersion
		}

		if len(lb.TLSConfig.ALPN) > 0 {
			tlsData["ALPN"] = lb.TLSConfig.ALPN
		}

		data["TLSConfig"] = tlsData
	}

	// Add timeouts if configured
	if lb.Timeouts != nil {
		data["Timeouts"] = map[string]int{
			"Idle":    lb.Timeouts.Idle,
			"Request": lb.Timeouts.Request,
		}
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute listener template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateCluster generates an Envoy cluster configuration
func (g *Generator) GenerateCluster(lb *models.LoadBalancer) ([]byte, error) {
	tmpl, err := template.New("cluster").Parse(clusterTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cluster template: %w", err)
	}

	// Validate and prepare endpoints
	endpoints := make([]map[string]interface{}, 0, len(lb.Backends))
	for _, backend := range lb.Backends {
		if !backend.Enabled {
			continue
		}

		// Validate backend address to prevent template injection
		if addrErr := validateAddress(backend.Address); addrErr != nil {
			return nil, fmt.Errorf("invalid backend address for %s: %w", backend.ID, addrErr)
		}

		ep := map[string]interface{}{
			"Address": backend.Address,
			"Port":    backend.Port,
		}

		if backend.Weight > 0 {
			ep["Weight"] = backend.Weight
		}

		endpoints = append(endpoints, ep)
	}

	// Prepare template data
	data := map[string]interface{}{
		"Name":              fmt.Sprintf("cluster_%s", lb.ID),
		"ConnectTimeout":    5,
		"LoadBalancingAlgo": string(lb.Algorithm),
		"Endpoints":         endpoints,
	}

	// Validate and add health check config
	if lb.HealthCheck != nil {
		if lb.HealthCheck.IsHTTPBased() {
			if pathErr := validateHealthCheckPath(lb.HealthCheck.Path); pathErr != nil {
				return nil, fmt.Errorf("invalid health check config: %w", pathErr)
			}
		}
		hcData := map[string]interface{}{
			"Type":               string(lb.HealthCheck.Type),
			"Timeout":            lb.HealthCheck.Timeout,
			"Interval":           lb.HealthCheck.Interval,
			"UnhealthyThreshold": lb.HealthCheck.UnhealthyThreshold,
			"HealthyThreshold":   lb.HealthCheck.HealthyThreshold,
		}

		if lb.HealthCheck.IsHTTPBased() {
			hcData["Path"] = lb.HealthCheck.Path
			if len(lb.HealthCheck.ExpectedStatus) > 0 {
				hcData["ExpectedStatus"] = lb.HealthCheck.ExpectedStatus
			}
		}

		data["HealthCheck"] = hcData
	}

	// Add circuit breakers
	data["CircuitBreakers"] = map[string]int{
		"MaxConnections":     1024,
		"MaxPendingRequests": 1024,
		"MaxRequests":        1024,
		"MaxRetries":         3,
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute cluster template: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateFullConfig generates complete Envoy configuration (listeners + clusters)
func (g *Generator) GenerateFullConfig(lb *models.LoadBalancer) (*EnvoyConfig, error) {
	// Validate load balancer config
	if err := lb.Validate(); err != nil {
		return nil, fmt.Errorf("invalid load balancer config: %w", err)
	}

	// Generate listener
	listenerYAML, err := g.GenerateListener(lb)
	if err != nil {
		return nil, fmt.Errorf("failed to generate listener: %w", err)
	}

	// Generate cluster
	clusterYAML, err := g.GenerateCluster(lb)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cluster: %w", err)
	}

	// Parse YAML to ensure it's valid
	var listenerData, clusterData interface{}
	if err = yaml.Unmarshal(listenerYAML, &listenerData); err != nil {
		return nil, fmt.Errorf("invalid listener YAML: %w", err)
	}
	if err = yaml.Unmarshal(clusterYAML, &clusterData); err != nil {
		return nil, fmt.Errorf("invalid cluster YAML: %w", err)
	}

	return &EnvoyConfig{
		Listeners: listenerYAML,
		Clusters:  clusterYAML,
	}, nil
}

// EnvoyConfig represents the generated Envoy configuration
type EnvoyConfig struct {
	Listeners []byte
	Clusters  []byte
}
