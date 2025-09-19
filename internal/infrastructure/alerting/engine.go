package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/sirupsen/logrus"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertState represents the state of an alert
type AlertState string

const (
	StateInactive AlertState = "inactive"
	StatePending  AlertState = "pending"
	StateFiring   AlertState = "firing"
	StateResolved AlertState = "resolved"
)

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	MetricType  string                 `json:"metricType" yaml:"metricType"`
	Threshold   float64                `json:"threshold" yaml:"threshold"`
	Operator    string                 `json:"operator" yaml:"operator"` // gt, gte, lt, lte, eq, ne
	Duration    time.Duration          `json:"duration" yaml:"duration"`
	Severity    AlertSeverity          `json:"severity" yaml:"severity"`
	Labels      map[string]string      `json:"labels" yaml:"labels"`
	Annotations map[string]string      `json:"annotations" yaml:"annotations"`
	Channels    []string               `json:"channels" yaml:"channels"`
	Enabled     bool                   `json:"enabled" yaml:"enabled"`
}

// Alert represents an active alert
type Alert struct {
	ID          string            `json:"id"`
	RuleID      string            `json:"ruleId"`
	RuleName    string            `json:"ruleName"`
	State       AlertState        `json:"state"`
	Severity    AlertSeverity     `json:"severity"`
	Message     string            `json:"message"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      *time.Time        `json:"endsAt,omitempty"`
	LastEvalAt  time.Time         `json:"lastEvalAt"`
	Count       int               `json:"count"`
}

// AlertingEngine manages alert rules and evaluation
type AlertingEngine struct {
	mu              sync.RWMutex
	rules           map[string]*AlertRule
	activeAlerts    map[string]*Alert
	alertHistory    []*Alert
	notifier        *NotificationManager
	config          AlertingConfig
	logger          *logrus.Logger
	evaluationTicker *time.Ticker
	stopCh          chan struct{}
	rateLimiter     map[string]time.Time
}

// AlertingConfig holds configuration for the alerting engine
type AlertingConfig struct {
	Enabled            bool          `json:"enabled" yaml:"enabled"`
	EvaluationInterval time.Duration `json:"evaluationInterval" yaml:"evaluationInterval"`
	RateLimitWindow    time.Duration `json:"rateLimitWindow" yaml:"rateLimitWindow"`
	MaxHistorySize     int           `json:"maxHistorySize" yaml:"maxHistorySize"`
	DefaultSeverity    AlertSeverity `json:"defaultSeverity" yaml:"defaultSeverity"`
}

// DefaultAlertingConfig returns default alerting configuration
func DefaultAlertingConfig() AlertingConfig {
	return AlertingConfig{
		Enabled:            true,
		EvaluationInterval: 30 * time.Second,
		RateLimitWindow:    5 * time.Minute,
		MaxHistorySize:     1000,
		DefaultSeverity:    SeverityWarning,
	}
}

// NewAlertingEngine creates a new alerting engine
func NewAlertingEngine(config AlertingConfig, notifier *NotificationManager, logger *logrus.Logger) *AlertingEngine {
	return &AlertingEngine{
		rules:        make(map[string]*AlertRule),
		activeAlerts: make(map[string]*Alert),
		alertHistory: make([]*Alert, 0),
		notifier:     notifier,
		config:       config,
		logger:       logger,
		stopCh:       make(chan struct{}),
		rateLimiter:  make(map[string]time.Time),
	}
}

// AddRule adds a new alert rule
func (e *AlertingEngine) AddRule(rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	if rule.Name == "" {
		rule.Name = rule.ID
	}

	if rule.Severity == "" {
		rule.Severity = e.config.DefaultSeverity
	}

	e.rules[rule.ID] = rule
	e.logger.WithFields(logrus.Fields{
		"ruleID":   rule.ID,
		"ruleName": rule.Name,
	}).Info("Alert rule added")

	return nil
}

// RemoveRule removes an alert rule
func (e *AlertingEngine) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	delete(e.rules, ruleID)

	// Remove any active alerts for this rule
	for alertID, alert := range e.activeAlerts {
		if alert.RuleID == ruleID {
			alert.State = StateResolved
			alert.EndsAt = &[]time.Time{time.Now()}[0]
			e.alertHistory = append(e.alertHistory, alert)
			delete(e.activeAlerts, alertID)
		}
	}

	e.logger.WithField("ruleID", ruleID).Info("Alert rule removed")
	return nil
}

// GetRule returns a specific alert rule
func (e *AlertingEngine) GetRule(ruleID string) (*AlertRule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rule, exists := e.rules[ruleID]
	return rule, exists
}

// GetAllRules returns all alert rules
func (e *AlertingEngine) GetAllRules() []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		rules = append(rules, rule)
	}
	return rules
}

// EvaluateSystemMetrics evaluates alert rules against system metrics
func (e *AlertingEngine) EvaluateSystemMetrics(metrics *entities.SystemMetrics) {
	if metrics == nil || !e.config.Enabled {
		return
	}

	e.mu.RLock()
	rules := make([]*AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		e.evaluateSystemRule(rule, metrics)
	}
}

// EvaluateProcessMetrics evaluates alert rules against process metrics
func (e *AlertingEngine) EvaluateProcessMetrics(metrics []*entities.ProcessMetrics) {
	if metrics == nil || !e.config.Enabled {
		return
	}

	e.mu.RLock()
	rules := make([]*AlertRule, 0, len(e.rules))
	for _, rule := range e.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	e.mu.RUnlock()

	for _, rule := range rules {
		e.evaluateProcessRule(rule, metrics)
	}
}

// evaluateSystemRule evaluates a single rule against system metrics
func (e *AlertingEngine) evaluateSystemRule(rule *AlertRule, metrics *entities.SystemMetrics) {
	var value float64
	var shouldAlert bool

	switch rule.MetricType {
	case "cpu_usage":
		value = metrics.CPUUsage.Total()
	case "memory_usage":
		value = float64(metrics.MemoryUsage.UsedPercent())
	case "memory_available":
		value = float64(metrics.MemoryUsage.Available())
	case "load_average_1":
		value = metrics.LoadAverage.Values()[0]
	case "load_average_5":
		value = metrics.LoadAverage.Values()[1]
	case "load_average_15":
		value = metrics.LoadAverage.Values()[2]
	case "swap_usage":
		value = float64(metrics.SwapUsage.UsedPercent())
	default:
		return
	}

	shouldAlert = e.evaluateThreshold(value, rule.Threshold, rule.Operator)

	alertID := fmt.Sprintf("%s_%s", rule.ID, rule.MetricType)
	e.processAlertState(alertID, rule, value, shouldAlert)
}

// evaluateProcessRule evaluates a single rule against process metrics
func (e *AlertingEngine) evaluateProcessRule(rule *AlertRule, metrics []*entities.ProcessMetrics) {
	for _, process := range metrics {
		var value float64
		var shouldAlert bool

		switch rule.MetricType {
		case "process_cpu":
			value = process.CPUPercent.Value()
		case "process_memory":
			value = float64(process.MemoryPercent.Value())
		case "process_threads":
			value = float64(process.NumThreads.Value())
		case "process_open_files":
			value = float64(process.OpenFiles.Value())
		default:
			continue
		}

		shouldAlert = e.evaluateThreshold(value, rule.Threshold, rule.Operator)

		alertID := fmt.Sprintf("%s_%s_%d", rule.ID, rule.MetricType, process.PID.Value())

		// Add process information to labels
		labels := make(map[string]string)
		for k, v := range rule.Labels {
			labels[k] = v
		}
		labels["pid"] = fmt.Sprintf("%d", process.PID.Value())
		labels["process_name"] = process.Name.Value()
		labels["user"] = process.User.Value()

		e.processAlertStateWithLabels(alertID, rule, value, shouldAlert, labels)
	}
}

// evaluateThreshold evaluates if a value meets the threshold condition
func (e *AlertingEngine) evaluateThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case "gt", ">":
		return value > threshold
	case "gte", ">=":
		return value >= threshold
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	case "eq", "==":
		return value == threshold
	case "ne", "!=":
		return value != threshold
	default:
		return false
	}
}

// processAlertState processes the alert state based on evaluation
func (e *AlertingEngine) processAlertState(alertID string, rule *AlertRule, value float64, shouldAlert bool) {
	e.processAlertStateWithLabels(alertID, rule, value, shouldAlert, rule.Labels)
}

// processAlertStateWithLabels processes the alert state with custom labels
func (e *AlertingEngine) processAlertStateWithLabels(alertID string, rule *AlertRule, value float64, shouldAlert bool, labels map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	alert, exists := e.activeAlerts[alertID]
	now := time.Now()

	if shouldAlert {
		if !exists {
			// Create new alert
			alert = &Alert{
				ID:          alertID,
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				State:       StatePending,
				Severity:    rule.Severity,
				Message:     e.formatAlertMessage(rule, value),
				Value:       value,
				Threshold:   rule.Threshold,
				Labels:      labels,
				Annotations: rule.Annotations,
				StartsAt:    now,
				LastEvalAt:  now,
				Count:       1,
			}
			e.activeAlerts[alertID] = alert

			// Check if duration requirement is met
			if rule.Duration == 0 {
				e.fireAlert(alert)
			}
		} else {
			// Update existing alert
			alert.Value = value
			alert.LastEvalAt = now
			alert.Count++
			alert.Message = e.formatAlertMessage(rule, value)

			// Check if pending alert should fire
			if alert.State == StatePending && now.Sub(alert.StartsAt) >= rule.Duration {
				e.fireAlert(alert)
			}
		}
	} else if exists {
		// Resolve alert
		alert.State = StateResolved
		alert.EndsAt = &now
		e.alertHistory = append(e.alertHistory, alert)
		delete(e.activeAlerts, alertID)

		// Send resolution notification
		e.sendNotification(alert, true)

		e.logger.WithFields(logrus.Fields{
			"alertID": alertID,
			"rule":    rule.Name,
		}).Info("Alert resolved")
	}

	// Trim history if needed
	if len(e.alertHistory) > e.config.MaxHistorySize {
		e.alertHistory = e.alertHistory[len(e.alertHistory)-e.config.MaxHistorySize:]
	}
}

// fireAlert transitions an alert to firing state and sends notifications
func (e *AlertingEngine) fireAlert(alert *Alert) {
	alert.State = StateFiring

	// Check rate limiting
	if !e.shouldSendNotification(alert.ID) {
		return
	}

	// Send notification
	e.sendNotification(alert, false)

	e.logger.WithFields(logrus.Fields{
		"alertID":  alert.ID,
		"rule":     alert.RuleName,
		"severity": alert.Severity,
		"value":    alert.Value,
	}).Warn("Alert firing")
}

// shouldSendNotification checks if notification should be sent based on rate limiting
func (e *AlertingEngine) shouldSendNotification(alertID string) bool {
	lastSent, exists := e.rateLimiter[alertID]
	now := time.Now()

	if !exists || now.Sub(lastSent) >= e.config.RateLimitWindow {
		e.rateLimiter[alertID] = now
		return true
	}

	return false
}

// sendNotification sends alert notification through configured channels
func (e *AlertingEngine) sendNotification(alert *Alert, isResolution bool) {
	if e.notifier == nil {
		return
	}

	// Get rule to find notification channels
	rule, exists := e.rules[alert.RuleID]
	if !exists || len(rule.Channels) == 0 {
		return
	}

	notification := &Notification{
		Alert:        alert,
		IsResolution: isResolution,
		Timestamp:    time.Now(),
	}

	for _, channel := range rule.Channels {
		if err := e.notifier.Send(channel, notification); err != nil {
			e.logger.WithError(err).WithField("channel", channel).Error("Failed to send notification")
		}
	}
}

// formatAlertMessage formats the alert message
func (e *AlertingEngine) formatAlertMessage(rule *AlertRule, value float64) string {
	return fmt.Sprintf("%s: %.2f %s %.2f", rule.Description, value, rule.Operator, rule.Threshold)
}

// GetActiveAlerts returns all active alerts
func (e *AlertingEngine) GetActiveAlerts() []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alerts := make([]*Alert, 0, len(e.activeAlerts))
	for _, alert := range e.activeAlerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetAlertHistory returns the alert history
func (e *AlertingEngine) GetAlertHistory(limit int) []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 || limit > len(e.alertHistory) {
		limit = len(e.alertHistory)
	}

	start := len(e.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	return e.alertHistory[start:]
}

// Start starts the alerting engine evaluation loop
func (e *AlertingEngine) Start(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	e.evaluationTicker = time.NewTicker(e.config.EvaluationInterval)

	go func() {
		for {
			select {
			case <-e.evaluationTicker.C:
				// Periodic evaluation would happen here with latest metrics
			case <-e.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	e.logger.Info("Alerting engine started")
	return nil
}

// Stop stops the alerting engine
func (e *AlertingEngine) Stop(ctx context.Context) error {
	if e.evaluationTicker != nil {
		e.evaluationTicker.Stop()
	}

	select {
	case e.stopCh <- struct{}{}:
	default:
	}

	e.logger.Info("Alerting engine stopped")
	return nil
}

// LoadRules loads alert rules from a slice
func (e *AlertingEngine) LoadRules(rules []*AlertRule) error {
	for _, rule := range rules {
		if err := e.AddRule(rule); err != nil {
			return fmt.Errorf("failed to add rule %s: %w", rule.ID, err)
		}
	}
	return nil
}

// GetStatistics returns alerting engine statistics
func (e *AlertingEngine) GetStatistics() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	stats := map[string]interface{}{
		"totalRules":    len(e.rules),
		"enabledRules":  0,
		"activeAlerts":  len(e.activeAlerts),
		"historySize":   len(e.alertHistory),
		"firingAlerts":  0,
		"pendingAlerts": 0,
	}

	for _, rule := range e.rules {
		if rule.Enabled {
			stats["enabledRules"] = stats["enabledRules"].(int) + 1
		}
	}

	for _, alert := range e.activeAlerts {
		switch alert.State {
		case StateFiring:
			stats["firingAlerts"] = stats["firingAlerts"].(int) + 1
		case StatePending:
			stats["pendingAlerts"] = stats["pendingAlerts"].(int) + 1
		}
	}

	return stats
}