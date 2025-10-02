package monitoring

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// MetricsCollector collects and reports application metrics
type MetricsCollector struct {
	logger  zerolog.Logger
	config  MetricsConfig
	metrics map[string]*Metric
	mutex   sync.RWMutex

	// Performance tracking
	requestDurations map[string][]time.Duration
	requestCounts    map[string]int64
	errorCounts      map[string]int64

	// System metrics
	systemMetrics *SystemMetrics

	// Alert thresholds
	alertThresholds map[string]AlertThreshold
}

// MetricsConfig defines monitoring configuration
type MetricsConfig struct {
	ServiceName        string
	Environment        string
	CollectionInterval time.Duration
	RetentionPeriod    time.Duration

	// Performance thresholds
	ResponseTimeThreshold time.Duration
	ErrorRateThreshold    float64

	// Alert configuration
	AlertingEnabled bool
	AlertWebhookURL string
	SlackWebhookURL string
	EmailAlerts     []string

	// Metrics export
	PrometheusEnabled  bool
	PrometheusEndpoint string
	DatadogEnabled     bool
	DatadogAPIKey      string
}

// Metric represents a single metric
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description"`
}

// MetricType defines the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// SystemMetrics tracks system-level metrics
type SystemMetrics struct {
	CPUUsage        float64   `json:"cpu_usage"`
	MemoryUsage     float64   `json:"memory_usage"`
	DiskUsage       float64   `json:"disk_usage"`
	NetworkBytesIn  int64     `json:"network_bytes_in"`
	NetworkBytesOut int64     `json:"network_bytes_out"`
	GoroutineCount  int       `json:"goroutine_count"`
	HeapSize        int64     `json:"heap_size"`
	GCPauses        []float64 `json:"gc_pauses"`
	LastUpdated     time.Time `json:"last_updated"`
}

// AlertThreshold defines thresholds for alerting
type AlertThreshold struct {
	MetricName    string        `json:"metric_name"`
	Operator      string        `json:"operator"` // >, <, >=, <=, ==, !=
	Threshold     float64       `json:"threshold"`
	Duration      time.Duration `json:"duration"` // How long condition must persist
	Severity      AlertSeverity `json:"severity"`
	Description   string        `json:"description"`
	LastTriggered time.Time     `json:"last_triggered"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a triggered alert
type Alert struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	MetricName   string            `json:"metric_name"`
	CurrentValue float64           `json:"current_value"`
	Threshold    float64           `json:"threshold"`
	Severity     AlertSeverity     `json:"severity"`
	Description  string            `json:"description"`
	Service      string            `json:"service"`
	Environment  string            `json:"environment"`
	Labels       map[string]string `json:"labels"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger zerolog.Logger, config MetricsConfig) *MetricsCollector {
	mc := &MetricsCollector{
		logger:           logger,
		config:           config,
		metrics:          make(map[string]*Metric),
		requestDurations: make(map[string][]time.Duration),
		requestCounts:    make(map[string]int64),
		errorCounts:      make(map[string]int64),
		systemMetrics:    &SystemMetrics{},
		alertThresholds:  make(map[string]AlertThreshold),
	}

	// Initialize default alert thresholds
	mc.initializeDefaultThresholds()

	// Start background collection
	go mc.startCollection()

	return mc
}

// RecordRequestDuration records the duration of a request
func (mc *MetricsCollector) RecordRequestDuration(method string, duration time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.requestDurations[method] = append(mc.requestDurations[method], duration)
	mc.requestCounts[method]++

	// Keep only recent durations to prevent memory bloat
	if len(mc.requestDurations[method]) > 1000 {
		mc.requestDurations[method] = mc.requestDurations[method][len(mc.requestDurations[method])-500:]
	}

	// Update metrics
	mc.updateMetric("request_duration_seconds", float64(duration.Seconds()), map[string]string{
		"method": method,
	}, MetricTypeHistogram)

	mc.updateMetric("request_count_total", float64(mc.requestCounts[method]), map[string]string{
		"method": method,
	}, MetricTypeCounter)
}

// RecordError records an error occurrence
func (mc *MetricsCollector) RecordError(method string, errorType string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := method + ":" + errorType
	mc.errorCounts[key]++

	mc.updateMetric("error_count_total", float64(mc.errorCounts[key]), map[string]string{
		"method":     method,
		"error_type": errorType,
	}, MetricTypeCounter)
}

// RecordCustomMetric records a custom metric
func (mc *MetricsCollector) RecordCustomMetric(name string, value float64, labels map[string]string, metricType MetricType) {
	mc.updateMetric(name, value, labels, metricType)
}

// updateMetric updates or creates a metric
func (mc *MetricsCollector) updateMetric(name string, value float64, labels map[string]string, metricType MetricType) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.buildMetricKey(name, labels)

	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      metricType,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// buildMetricKey builds a unique key for a metric
func (mc *MetricsCollector) buildMetricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += ":" + k + "=" + v
	}
	return key
}

// GetMetrics returns all current metrics
func (mc *MetricsCollector) GetMetrics() map[string]*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*Metric)
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

// GetSystemMetrics returns current system metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return mc.systemMetrics
}

// startCollection starts the background metrics collection
func (mc *MetricsCollector) startCollection() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		mc.collectSystemMetrics()
		mc.checkAlertThresholds()
		mc.exportMetrics()
	}
}

// collectSystemMetrics collects system-level metrics
func (mc *MetricsCollector) collectSystemMetrics() {
	// This is a simplified implementation
	// In production, you would use proper system monitoring libraries

	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.systemMetrics.LastUpdated = time.Now()

	// Update system metrics in the main metrics map
	mc.metrics["system_cpu_usage"] = &Metric{
		Name:      "system_cpu_usage",
		Type:      MetricTypeGauge,
		Value:     mc.systemMetrics.CPUUsage,
		Timestamp: time.Now(),
	}

	mc.metrics["system_memory_usage"] = &Metric{
		Name:      "system_memory_usage",
		Type:      MetricTypeGauge,
		Value:     mc.systemMetrics.MemoryUsage,
		Timestamp: time.Now(),
	}

	mc.metrics["system_goroutine_count"] = &Metric{
		Name:      "system_goroutine_count",
		Type:      MetricTypeGauge,
		Value:     float64(mc.systemMetrics.GoroutineCount),
		Timestamp: time.Now(),
	}
}

// checkAlertThresholds checks if any alert thresholds are breached
func (mc *MetricsCollector) checkAlertThresholds() {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	for _, threshold := range mc.alertThresholds {
		if metric, exists := mc.metrics[threshold.MetricName]; exists {
			if mc.evaluateThreshold(metric.Value, threshold) {
				mc.triggerAlert(threshold, metric.Value)
			}
		}
	}
}

// evaluateThreshold evaluates if a threshold condition is met
func (mc *MetricsCollector) evaluateThreshold(value float64, threshold AlertThreshold) bool {
	switch threshold.Operator {
	case ">":
		return value > threshold.Threshold
	case "<":
		return value < threshold.Threshold
	case ">=":
		return value >= threshold.Threshold
	case "<=":
		return value <= threshold.Threshold
	case "==":
		return value == threshold.Threshold
	case "!=":
		return value != threshold.Threshold
	default:
		return false
	}
}

// triggerAlert triggers an alert
func (mc *MetricsCollector) triggerAlert(threshold AlertThreshold, currentValue float64) {
	// Avoid spam - don't trigger if recently triggered
	if time.Since(threshold.LastTriggered) < time.Minute*5 {
		return
	}

	alert := Alert{
		ID:           generateAlertID(),
		Timestamp:    time.Now(),
		MetricName:   threshold.MetricName,
		CurrentValue: currentValue,
		Threshold:    threshold.Threshold,
		Severity:     threshold.Severity,
		Description:  threshold.Description,
		Service:      mc.config.ServiceName,
		Environment:  mc.config.Environment,
	}

	// Log alert
	mc.logger.Error().
		Str("alert_id", alert.ID).
		Str("metric_name", alert.MetricName).
		Float64("current_value", alert.CurrentValue).
		Float64("threshold", alert.Threshold).
		Str("severity", string(alert.Severity)).
		Msg("Alert triggered")

	// Send alert notifications
	if mc.config.AlertingEnabled {
		go mc.sendAlertNotifications(alert)
	}

	// Update last triggered time
	mc.mutex.Lock()
	threshold.LastTriggered = time.Now()
	mc.alertThresholds[threshold.MetricName] = threshold
	mc.mutex.Unlock()
}

// sendAlertNotifications sends alert notifications
func (mc *MetricsCollector) sendAlertNotifications(alert Alert) {
	// Notification implementation can be added based on requirements:
	// - Webhook notifications
	// - Slack integration
	// - Email alerts
	mc.logger.Info().
		Str("alert_id", alert.ID).
		Msg("Sending alert notifications")
}

// exportMetrics exports metrics to external systems
func (mc *MetricsCollector) exportMetrics() {
	if mc.config.PrometheusEnabled {
		mc.exportToPrometheus()
	}

	if mc.config.DatadogEnabled {
		mc.exportToDatadog()
	}
}

// exportToPrometheus exports metrics to Prometheus
func (mc *MetricsCollector) exportToPrometheus() {
	// Placeholder - implement Prometheus exposition format
	mc.logger.Debug().Msg("Exporting metrics to Prometheus")
}

// exportToDatadog exports metrics to Datadog
func (mc *MetricsCollector) exportToDatadog() {
	// Placeholder - implement Datadog API integration
	mc.logger.Debug().Msg("Exporting metrics to Datadog")
}

// initializeDefaultThresholds sets up default alert thresholds
func (mc *MetricsCollector) initializeDefaultThresholds() {
	mc.alertThresholds["request_duration_seconds"] = AlertThreshold{
		MetricName:  "request_duration_seconds",
		Operator:    ">",
		Threshold:   5.0, // 5 seconds
		Duration:    time.Minute,
		Severity:    AlertSeverityMedium,
		Description: "Request duration is too high",
	}

	mc.alertThresholds["error_rate"] = AlertThreshold{
		MetricName:  "error_rate",
		Operator:    ">",
		Threshold:   0.05, // 5% error rate
		Duration:    time.Minute * 5,
		Severity:    AlertSeverityHigh,
		Description: "Error rate is too high",
	}

	mc.alertThresholds["system_cpu_usage"] = AlertThreshold{
		MetricName:  "system_cpu_usage",
		Operator:    ">",
		Threshold:   0.8, // 80% CPU usage
		Duration:    time.Minute * 2,
		Severity:    AlertSeverityHigh,
		Description: "CPU usage is too high",
	}

	mc.alertThresholds["system_memory_usage"] = AlertThreshold{
		MetricName:  "system_memory_usage",
		Operator:    ">",
		Threshold:   0.9, // 90% memory usage
		Duration:    time.Minute * 2,
		Severity:    AlertSeverityCritical,
		Description: "Memory usage is critically high",
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return "alert_" + string(rune(time.Now().UnixNano()))
}

// GetDefaultMetricsConfig returns default metrics configuration
func GetDefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		ServiceName:           "klubbspel-api",
		Environment:           "production",
		CollectionInterval:    time.Second * 30,
		RetentionPeriod:       time.Hour * 24 * 7, // 7 days
		ResponseTimeThreshold: time.Second * 2,
		ErrorRateThreshold:    0.05, // 5%

		AlertingEnabled: true,

		PrometheusEnabled:  false,
		PrometheusEndpoint: "/metrics",
		DatadogEnabled:     false,
	}
}
