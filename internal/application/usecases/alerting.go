package usecases

import (
	"context"
	"fmt"

	"github.com/reinaldosaraiva/goiotop/internal/domain/entities"
	"github.com/reinaldosaraiva/goiotop/internal/infrastructure/alerting"
	"github.com/sirupsen/logrus"
)

// AlertingUseCase handles alerting logic
type AlertingUseCase struct {
	engine *alerting.AlertingEngine
	logger *logrus.Logger
}

// NewAlertingUseCase creates a new alerting use case
func NewAlertingUseCase(engine *alerting.AlertingEngine, logger *logrus.Logger) *AlertingUseCase {
	return &AlertingUseCase{
		engine: engine,
		logger: logger,
	}
}

// EvaluateSystemMetrics evaluates system metrics for alerts
func (u *AlertingUseCase) EvaluateSystemMetrics(metrics *entities.SystemMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	u.engine.EvaluateSystemMetrics(metrics)
	return nil
}

// EvaluateProcessMetrics evaluates process metrics for alerts
func (u *AlertingUseCase) EvaluateProcessMetrics(metrics []*entities.ProcessMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	u.engine.EvaluateProcessMetrics(metrics)
	return nil
}

// AddAlertRule adds a new alert rule
func (u *AlertingUseCase) AddAlertRule(rule *alerting.AlertRule) error {
	return u.engine.AddRule(rule)
}

// RemoveAlertRule removes an alert rule
func (u *AlertingUseCase) RemoveAlertRule(ruleID string) error {
	return u.engine.RemoveRule(ruleID)
}

// GetAlertRule returns a specific alert rule
func (u *AlertingUseCase) GetAlertRule(ruleID string) (*alerting.AlertRule, error) {
	rule, exists := u.engine.GetRule(ruleID)
	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}
	return rule, nil
}

// GetAllAlertRules returns all alert rules
func (u *AlertingUseCase) GetAllAlertRules() []*alerting.AlertRule {
	return u.engine.GetAllRules()
}

// GetActiveAlerts returns all active alerts
func (u *AlertingUseCase) GetActiveAlerts() []*alerting.Alert {
	return u.engine.GetActiveAlerts()
}

// GetAlertHistory returns alert history
func (u *AlertingUseCase) GetAlertHistory(limit int) []*alerting.Alert {
	return u.engine.GetAlertHistory(limit)
}

// GetStatistics returns alerting statistics
func (u *AlertingUseCase) GetStatistics() map[string]interface{} {
	return u.engine.GetStatistics()
}

// Start starts the alerting engine
func (u *AlertingUseCase) Start(ctx context.Context) error {
	return u.engine.Start(ctx)
}

// Stop stops the alerting engine
func (u *AlertingUseCase) Stop(ctx context.Context) error {
	return u.engine.Stop(ctx)
}

// LoadRulesFromConfig loads alert rules from configuration
func (u *AlertingUseCase) LoadRulesFromConfig(rules []alerting.AlertRule) error {
	rulesPtr := make([]*alerting.AlertRule, len(rules))
	for i := range rules {
		rulesPtr[i] = &rules[i]
	}
	return u.engine.LoadRules(rulesPtr)
}

// TestAlertRule tests an alert rule with sample data
func (u *AlertingUseCase) TestAlertRule(ruleID string, sampleValue float64) (bool, string) {
	rule, exists := u.engine.GetRule(ruleID)
	if !exists {
		return false, fmt.Sprintf("Rule %s not found", ruleID)
	}

	var shouldAlert bool
	switch rule.Operator {
	case "gt", ">":
		shouldAlert = sampleValue > rule.Threshold
	case "gte", ">=":
		shouldAlert = sampleValue >= rule.Threshold
	case "lt", "<":
		shouldAlert = sampleValue < rule.Threshold
	case "lte", "<=":
		shouldAlert = sampleValue <= rule.Threshold
	case "eq", "==":
		shouldAlert = sampleValue == rule.Threshold
	case "ne", "!=":
		shouldAlert = sampleValue != rule.Threshold
	default:
		return false, fmt.Sprintf("Unknown operator: %s", rule.Operator)
	}

	if shouldAlert {
		return true, fmt.Sprintf("Alert would trigger: %.2f %s %.2f", sampleValue, rule.Operator, rule.Threshold)
	}
	return false, fmt.Sprintf("Alert would not trigger: %.2f %s %.2f", sampleValue, rule.Operator, rule.Threshold)
}

// SilenceAlert temporarily silences an alert
func (u *AlertingUseCase) SilenceAlert(alertID string, duration string) error {
	// This would implement alert silencing logic
	u.logger.WithFields(logrus.Fields{
		"alertID":  alertID,
		"duration": duration,
	}).Info("Alert silenced")
	return nil
}

// AcknowledgeAlert acknowledges an alert
func (u *AlertingUseCase) AcknowledgeAlert(alertID string, message string) error {
	// This would implement alert acknowledgement logic
	u.logger.WithFields(logrus.Fields{
		"alertID": alertID,
		"message": message,
	}).Info("Alert acknowledged")
	return nil
}