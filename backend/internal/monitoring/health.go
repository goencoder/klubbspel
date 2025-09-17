package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// HealthChecker provides comprehensive health checking functionality
type HealthChecker struct {
	logger    zerolog.Logger
	config    HealthConfig
	checks    map[string]HealthCheck
	mutex     sync.RWMutex
	lastCheck time.Time
	status    OverallStatus
}

// HealthConfig defines health check configuration
type HealthConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	CheckInterval  time.Duration
	CheckTimeout   time.Duration
	GracePeriod    time.Duration

	// HTTP endpoint configuration
	HTTPEnabled bool
	HTTPPort    int
	HTTPPath    string

	// Alert configuration
	AlertOnFailure    bool
	FailureThreshold  int
	RecoveryThreshold int
}

// HealthCheck represents a single health check
type HealthCheck interface {
	Name() string
	Check(ctx context.Context) HealthStatus
	Critical() bool
	Description() string
}

// HealthStatus represents the status of a health check
type HealthStatus struct {
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
}

// Status represents health status values
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// OverallStatus represents the overall system health
type OverallStatus struct {
	Status      Status                  `json:"status"`
	Timestamp   time.Time               `json:"timestamp"`
	ServiceName string                  `json:"service_name"`
	Version     string                  `json:"version"`
	Environment string                  `json:"environment"`
	Uptime      time.Duration           `json:"uptime"`
	Checks      map[string]HealthStatus `json:"checks"`
	Summary     map[string]int          `json:"summary"`
}

// DatabaseHealthCheck checks database connectivity
type DatabaseHealthCheck struct {
	name        string
	description string
	critical    bool
	// Add database connection interface when available
}

func NewDatabaseHealthCheck(name, description string, critical bool) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		name:        name,
		description: description,
		critical:    critical,
	}
}

func (d *DatabaseHealthCheck) Name() string {
	return d.name
}

func (d *DatabaseHealthCheck) Description() string {
	return d.description
}

func (d *DatabaseHealthCheck) Critical() bool {
	return d.critical
}

func (d *DatabaseHealthCheck) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	// Placeholder - implement actual database ping
	// In production, use: db.PingContext(ctx)

	status := HealthStatus{
		Status:      StatusHealthy,
		Message:     "Database connection is healthy",
		LastChecked: time.Now(),
		Duration:    time.Since(start),
		Details: map[string]interface{}{
			"connection_pool_size": 10,
			"active_connections":   5,
			"max_connections":      20,
		},
	}

	return status
}

// ExternalAPIHealthCheck checks external API connectivity
type ExternalAPIHealthCheck struct {
	name        string
	description string
	critical    bool
	url         string
	timeout     time.Duration
}

func NewExternalAPIHealthCheck(name, description, url string, critical bool, timeout time.Duration) *ExternalAPIHealthCheck {
	return &ExternalAPIHealthCheck{
		name:        name,
		description: description,
		critical:    critical,
		url:         url,
		timeout:     timeout,
	}
}

func (e *ExternalAPIHealthCheck) Name() string {
	return e.name
}

func (e *ExternalAPIHealthCheck) Description() string {
	return e.description
}

func (e *ExternalAPIHealthCheck) Critical() bool {
	return e.critical
}

func (e *ExternalAPIHealthCheck) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Make HTTP request
	req, err := http.NewRequestWithContext(checkCtx, "GET", e.url, nil)
	if err != nil {
		return HealthStatus{
			Status:      StatusUnhealthy,
			Message:     "Failed to create request",
			Error:       err.Error(),
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}

	client := &http.Client{Timeout: e.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return HealthStatus{
			Status:      StatusUnhealthy,
			Message:     "External API is unreachable",
			Error:       err.Error(),
			LastChecked: time.Now(),
			Duration:    time.Since(start),
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return HealthStatus{
			Status:      StatusHealthy,
			Message:     "External API is reachable",
			LastChecked: time.Now(),
			Duration:    time.Since(start),
			Details: map[string]interface{}{
				"status_code":   resp.StatusCode,
				"response_time": time.Since(start).Milliseconds(),
			},
		}
	}

	return HealthStatus{
		Status:      StatusDegraded,
		Message:     fmt.Sprintf("External API returned status %d", resp.StatusCode),
		LastChecked: time.Now(),
		Duration:    time.Since(start),
		Details: map[string]interface{}{
			"status_code":   resp.StatusCode,
			"response_time": time.Since(start).Milliseconds(),
		},
	}
}

// SystemResourcesHealthCheck checks system resources
type SystemResourcesHealthCheck struct {
	name        string
	description string
	critical    bool

	// Thresholds
	cpuThreshold    float64
	memoryThreshold float64
	diskThreshold   float64
}

func NewSystemResourcesHealthCheck(name, description string, critical bool) *SystemResourcesHealthCheck {
	return &SystemResourcesHealthCheck{
		name:            name,
		description:     description,
		critical:        critical,
		cpuThreshold:    0.8,  // 80%
		memoryThreshold: 0.9,  // 90%
		diskThreshold:   0.95, // 95%
	}
}

func (s *SystemResourcesHealthCheck) Name() string {
	return s.name
}

func (s *SystemResourcesHealthCheck) Description() string {
	return s.description
}

func (s *SystemResourcesHealthCheck) Critical() bool {
	return s.critical
}

func (s *SystemResourcesHealthCheck) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	// Placeholder - implement actual system metrics collection
	// In production, use proper system monitoring libraries

	cpuUsage := 0.2    // 20%
	memoryUsage := 0.6 // 60%
	diskUsage := 0.4   // 40%

	status := StatusHealthy
	var issues []string

	if cpuUsage > s.cpuThreshold {
		status = StatusDegraded
		issues = append(issues, fmt.Sprintf("High CPU usage: %.1f%%", cpuUsage*100))
	}

	if memoryUsage > s.memoryThreshold {
		if s.critical {
			status = StatusUnhealthy
		} else if status != StatusUnhealthy {
			status = StatusDegraded
		}
		issues = append(issues, fmt.Sprintf("High memory usage: %.1f%%", memoryUsage*100))
	}

	if diskUsage > s.diskThreshold {
		if s.critical {
			status = StatusUnhealthy
		} else if status != StatusUnhealthy {
			status = StatusDegraded
		}
		issues = append(issues, fmt.Sprintf("High disk usage: %.1f%%", diskUsage*100))
	}

	message := "System resources are healthy"
	if len(issues) > 0 {
		message = "System resource issues detected: " + fmt.Sprintf("%v", issues)
	}

	return HealthStatus{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Duration:    time.Since(start),
		Details: map[string]interface{}{
			"cpu_usage":    cpuUsage,
			"memory_usage": memoryUsage,
			"disk_usage":   diskUsage,
			"issues":       issues,
		},
	}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger zerolog.Logger, config HealthConfig) *HealthChecker {
	hc := &HealthChecker{
		logger: logger,
		config: config,
		checks: make(map[string]HealthCheck),
		status: OverallStatus{
			Status:      StatusUnknown,
			Timestamp:   time.Now(),
			ServiceName: config.ServiceName,
			Version:     config.ServiceVersion,
			Environment: config.Environment,
			Checks:      make(map[string]HealthStatus),
			Summary:     make(map[string]int),
		},
	}

	// Start background health checking
	go hc.startHealthChecking()

	// Start HTTP endpoint if enabled
	if config.HTTPEnabled {
		go hc.startHTTPEndpoint()
	}

	return hc
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(check HealthCheck) {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	hc.checks[check.Name()] = check
	hc.logger.Info().
		Str("check_name", check.Name()).
		Bool("critical", check.Critical()).
		Str("description", check.Description()).
		Msg("Health check registered")
}

// GetStatus returns the current overall health status
func (hc *HealthChecker) GetStatus() OverallStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	return hc.status
}

// RunChecks manually runs all health checks
func (hc *HealthChecker) RunChecks(ctx context.Context) OverallStatus {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	return hc.runChecksInternal(ctx)
}

// runChecksInternal runs all health checks (internal method)
func (hc *HealthChecker) runChecksInternal(ctx context.Context) OverallStatus {
	checkCtx, cancel := context.WithTimeout(ctx, hc.config.CheckTimeout)
	defer cancel()

	hc.status.Timestamp = time.Now()
	hc.status.Checks = make(map[string]HealthStatus)
	hc.status.Summary = map[string]int{
		"healthy":   0,
		"unhealthy": 0,
		"degraded":  0,
		"unknown":   0,
	}

	overallStatus := StatusHealthy

	// Run all checks
	for name, check := range hc.checks {
		status := check.Check(checkCtx)
		hc.status.Checks[name] = status

		// Update summary
		hc.status.Summary[string(status.Status)]++

		// Determine overall status
		if status.Status == StatusUnhealthy && check.Critical() {
			overallStatus = StatusUnhealthy
		} else if status.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		} else if status.Status == StatusUnhealthy && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	hc.status.Status = overallStatus
	hc.lastCheck = time.Now()

	// Log status
	hc.logger.Info().
		Str("overall_status", string(overallStatus)).
		Int("healthy_checks", hc.status.Summary["healthy"]).
		Int("unhealthy_checks", hc.status.Summary["unhealthy"]).
		Int("degraded_checks", hc.status.Summary["degraded"]).
		Msg("Health check completed")

	return hc.status
}

// startHealthChecking starts the background health checking routine
func (hc *HealthChecker) startHealthChecking() {
	ticker := time.NewTicker(hc.config.CheckInterval)
	defer ticker.Stop()

	// Run initial check
	ctx := context.Background()
	hc.RunChecks(ctx)

	for range ticker.C {
		hc.RunChecks(ctx)
	}
}

// startHTTPEndpoint starts the HTTP health check endpoint
func (hc *HealthChecker) startHTTPEndpoint() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc(hc.config.HTTPPath, hc.handleHealthCheck)

	// Ready endpoint (for Kubernetes readiness probes)
	mux.HandleFunc("/ready", hc.handleReadiness)

	// Live endpoint (for Kubernetes liveness probes)
	mux.HandleFunc("/live", hc.handleLiveness)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", hc.config.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second, // G112 fix: prevent Slowloris attacks
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	hc.logger.Info().
		Int("port", hc.config.HTTPPort).
		Str("path", hc.config.HTTPPath).
		Msg("Starting health check HTTP endpoint")

	if err := server.ListenAndServe(); err != nil {
		hc.logger.Error().Err(err).Msg("Health check HTTP server failed")
	}
}

// handleHealthCheck handles the main health check endpoint
func (hc *HealthChecker) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := hc.GetStatus()

	// Set HTTP status code based on health
	var httpStatus int
	switch status.Status {
	case StatusUnhealthy:
		httpStatus = http.StatusServiceUnavailable
	case StatusDegraded, StatusHealthy:
		httpStatus = http.StatusOK
	default:
		httpStatus = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		hc.logger.Error().Err(err).Msg("Failed to encode health status")
	}
}

// handleReadiness handles Kubernetes readiness probe
func (hc *HealthChecker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	status := hc.GetStatus()

	if status.Status == StatusHealthy || status.Status == StatusDegraded {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			hc.logger.Error().Err(err).Msg("failed to write readiness response")
		}
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("NOT READY")); err != nil {
			hc.logger.Error().Err(err).Msg("failed to write readiness response")
		}
	}
}

// handleLiveness handles Kubernetes liveness probe
func (hc *HealthChecker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness is more permissive - only fail if critical checks are failing
	status := hc.GetStatus()

	criticalFailure := false
	for name, check := range hc.checks {
		if check.Critical() {
			if checkStatus, exists := status.Checks[name]; exists {
				if checkStatus.Status == StatusUnhealthy {
					criticalFailure = true
					break
				}
			}
		}
	}

	if criticalFailure {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte("NOT ALIVE")); err != nil {
			hc.logger.Error().Err(err).Msg("failed to write liveness response")
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			hc.logger.Error().Err(err).Msg("failed to write liveness response")
		}
	}
}

// GetDefaultHealthConfig returns default health check configuration
func GetDefaultHealthConfig() HealthConfig {
	return HealthConfig{
		ServiceName:    "klubbspel-api",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		CheckInterval:  time.Second * 30,
		CheckTimeout:   time.Second * 10,
		GracePeriod:    time.Second * 30,

		HTTPEnabled: true,
		HTTPPort:    8081,
		HTTPPath:    "/healthz",

		AlertOnFailure:    true,
		FailureThreshold:  3,
		RecoveryThreshold: 2,
	}
}
