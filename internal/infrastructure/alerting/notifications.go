package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/smtp"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Notification represents a notification to be sent
type Notification struct {
	Alert        *Alert    `json:"alert"`
	IsResolution bool      `json:"isResolution"`
	Timestamp    time.Time `json:"timestamp"`
}

// NotificationChannel interface for notification channels
type NotificationChannel interface {
	Send(notification *Notification) error
	GetType() string
	IsEnabled() bool
}

// NotificationManager manages notification channels
type NotificationManager struct {
	mu       sync.RWMutex
	channels map[string]NotificationChannel
	logger   *logrus.Logger
	config   NotificationConfig
}

// NotificationConfig holds configuration for notifications
type NotificationConfig struct {
	Enabled           bool                   `json:"enabled" yaml:"enabled"`
	RateLimitPerAlert int                    `json:"rateLimitPerAlert" yaml:"rateLimitPerAlert"`
	RetryAttempts     int                    `json:"retryAttempts" yaml:"retryAttempts"`
	RetryDelay        time.Duration          `json:"retryDelay" yaml:"retryDelay"`
	Templates         map[string]string      `json:"templates" yaml:"templates"`
	Channels          []ChannelConfig        `json:"channels" yaml:"channels"`
}

// ChannelConfig holds configuration for a specific channel
type ChannelConfig struct {
	Name    string                 `json:"name" yaml:"name"`
	Type    string                 `json:"type" yaml:"type"`
	Enabled bool                   `json:"enabled" yaml:"enabled"`
	Config  map[string]interface{} `json:"config" yaml:"config"`
}

// DefaultNotificationConfig returns default notification configuration
func DefaultNotificationConfig() NotificationConfig {
	return NotificationConfig{
		Enabled:           true,
		RateLimitPerAlert: 5,
		RetryAttempts:     3,
		RetryDelay:        5 * time.Second,
		Templates:         make(map[string]string),
		Channels:          []ChannelConfig{},
	}
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config NotificationConfig, logger *logrus.Logger) *NotificationManager {
	manager := &NotificationManager{
		channels: make(map[string]NotificationChannel),
		logger:   logger,
		config:   config,
	}

	// Initialize channels from config
	for _, channelConfig := range config.Channels {
		if err := manager.AddChannelFromConfig(channelConfig); err != nil {
			logger.WithError(err).WithField("channel", channelConfig.Name).Error("Failed to add notification channel")
		}
	}

	return manager
}

// AddChannel adds a notification channel
func (m *NotificationManager) AddChannel(name string, channel NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.channels[name]; exists {
		return fmt.Errorf("channel %s already exists", name)
	}

	m.channels[name] = channel
	m.logger.WithField("channel", name).Info("Notification channel added")
	return nil
}

// RemoveChannel removes a notification channel
func (m *NotificationManager) RemoveChannel(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.channels[name]; !exists {
		return fmt.Errorf("channel %s not found", name)
	}

	delete(m.channels, name)
	m.logger.WithField("channel", name).Info("Notification channel removed")
	return nil
}

// AddChannelFromConfig adds a channel from configuration
func (m *NotificationManager) AddChannelFromConfig(config ChannelConfig) error {
	var channel NotificationChannel
	var err error

	switch config.Type {
	case "email":
		channel, err = NewEmailChannel(config.Config, m.logger)
	case "webhook":
		channel, err = NewWebhookChannel(config.Config, m.logger)
	case "slack":
		channel, err = NewSlackChannel(config.Config, m.logger)
	case "console":
		channel, err = NewConsoleChannel(config.Config, m.logger)
	default:
		return fmt.Errorf("unknown channel type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create %s channel: %w", config.Type, err)
	}

	return m.AddChannel(config.Name, channel)
}

// Send sends a notification through a specific channel
func (m *NotificationManager) Send(channelName string, notification *Notification) error {
	m.mu.RLock()
	channel, exists := m.channels[channelName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	if !channel.IsEnabled() {
		return nil
	}

	// Retry logic
	var lastErr error
	for i := 0; i < m.config.RetryAttempts; i++ {
		if err := channel.Send(notification); err != nil {
			lastErr = err
			m.logger.WithError(err).WithFields(logrus.Fields{
				"channel": channelName,
				"attempt": i + 1,
			}).Warn("Failed to send notification")

			if i < m.config.RetryAttempts-1 {
				time.Sleep(m.config.RetryDelay)
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", m.config.RetryAttempts, lastErr)
}

// SendToAll sends a notification to all channels
func (m *NotificationManager) SendToAll(notification *Notification) map[string]error {
	m.mu.RLock()
	channelNames := make([]string, 0, len(m.channels))
	for name := range m.channels {
		channelNames = append(channelNames, name)
	}
	m.mu.RUnlock()

	results := make(map[string]error)
	for _, name := range channelNames {
		results[name] = m.Send(name, notification)
	}

	return results
}

// EmailChannel implements email notifications
type EmailChannel struct {
	config EmailConfig
	logger *logrus.Logger
}

// EmailConfig holds email channel configuration
type EmailConfig struct {
	Enabled    bool     `json:"enabled" yaml:"enabled"`
	SMTPHost   string   `json:"smtpHost" yaml:"smtpHost"`
	SMTPPort   int      `json:"smtpPort" yaml:"smtpPort"`
	Username   string   `json:"username" yaml:"username"`
	Password   string   `json:"password" yaml:"password"`
	From       string   `json:"from" yaml:"from"`
	To         []string `json:"to" yaml:"to"`
	Subject    string   `json:"subject" yaml:"subject"`
	TLS        bool     `json:"tls" yaml:"tls"`
}

// NewEmailChannel creates a new email notification channel
func NewEmailChannel(config map[string]interface{}, logger *logrus.Logger) (*EmailChannel, error) {
	// Convert map to EmailConfig
	data, _ := json.Marshal(config)
	var emailConfig EmailConfig
	if err := json.Unmarshal(data, &emailConfig); err != nil {
		return nil, err
	}

	return &EmailChannel{
		config: emailConfig,
		logger: logger,
	}, nil
}

// Send sends an email notification
func (c *EmailChannel) Send(notification *Notification) error {
	subject := c.config.Subject
	if subject == "" {
		if notification.IsResolution {
			subject = fmt.Sprintf("RESOLVED: %s", notification.Alert.RuleName)
		} else {
			subject = fmt.Sprintf("[%s] %s", notification.Alert.Severity, notification.Alert.RuleName)
		}
	}

	body := c.formatEmailBody(notification)

	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", c.config.SMTPHost, c.config.SMTPPort)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s",
		c.config.To[0], subject, body))

	return smtp.SendMail(addr, auth, c.config.From, c.config.To, msg)
}

// formatEmailBody formats the email body
func (c *EmailChannel) formatEmailBody(notification *Notification) string {
	var buf bytes.Buffer
	alert := notification.Alert

	if notification.IsResolution {
		buf.WriteString("Alert RESOLVED\n\n")
	} else {
		buf.WriteString(fmt.Sprintf("Alert %s\n\n", alert.State))
	}

	buf.WriteString(fmt.Sprintf("Rule: %s\n", alert.RuleName))
	buf.WriteString(fmt.Sprintf("Severity: %s\n", alert.Severity))
	buf.WriteString(fmt.Sprintf("Message: %s\n", alert.Message))
	buf.WriteString(fmt.Sprintf("Value: %.2f\n", alert.Value))
	buf.WriteString(fmt.Sprintf("Threshold: %.2f\n", alert.Threshold))
	buf.WriteString(fmt.Sprintf("Started: %s\n", alert.StartsAt.Format(time.RFC3339)))

	if alert.EndsAt != nil {
		buf.WriteString(fmt.Sprintf("Ended: %s\n", alert.EndsAt.Format(time.RFC3339)))
	}

	if len(alert.Labels) > 0 {
		buf.WriteString("\nLabels:\n")
		for k, v := range alert.Labels {
			buf.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	return buf.String()
}

// GetType returns the channel type
func (c *EmailChannel) GetType() string {
	return "email"
}

// IsEnabled returns whether the channel is enabled
func (c *EmailChannel) IsEnabled() bool {
	return c.config.Enabled
}

// WebhookChannel implements webhook notifications
type WebhookChannel struct {
	config WebhookConfig
	client *http.Client
	logger *logrus.Logger
}

// WebhookConfig holds webhook channel configuration
type WebhookConfig struct {
	Enabled bool              `json:"enabled" yaml:"enabled"`
	URL     string            `json:"url" yaml:"url"`
	Method  string            `json:"method" yaml:"method"`
	Headers map[string]string `json:"headers" yaml:"headers"`
	Timeout time.Duration     `json:"timeout" yaml:"timeout"`
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(config map[string]interface{}, logger *logrus.Logger) (*WebhookChannel, error) {
	data, _ := json.Marshal(config)
	var webhookConfig WebhookConfig
	if err := json.Unmarshal(data, &webhookConfig); err != nil {
		return nil, err
	}

	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST"
	}

	if webhookConfig.Timeout == 0 {
		webhookConfig.Timeout = 30 * time.Second
	}

	return &WebhookChannel{
		config: webhookConfig,
		client: &http.Client{
			Timeout: webhookConfig.Timeout,
		},
		logger: logger,
	}, nil
}

// Send sends a webhook notification
func (c *WebhookChannel) Send(notification *Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(c.config.Method, c.config.URL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (c *WebhookChannel) GetType() string {
	return "webhook"
}

// IsEnabled returns whether the channel is enabled
func (c *WebhookChannel) IsEnabled() bool {
	return c.config.Enabled
}

// SlackChannel implements Slack notifications
type SlackChannel struct {
	config SlackConfig
	client *http.Client
	logger *logrus.Logger
}

// SlackConfig holds Slack channel configuration
type SlackConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	WebhookURL string `json:"webhookUrl" yaml:"webhookUrl"`
	Channel    string `json:"channel" yaml:"channel"`
	Username   string `json:"username" yaml:"username"`
	IconEmoji  string `json:"iconEmoji" yaml:"iconEmoji"`
}

// NewSlackChannel creates a new Slack notification channel
func NewSlackChannel(config map[string]interface{}, logger *logrus.Logger) (*SlackChannel, error) {
	data, _ := json.Marshal(config)
	var slackConfig SlackConfig
	if err := json.Unmarshal(data, &slackConfig); err != nil {
		return nil, err
	}

	return &SlackChannel{
		config: slackConfig,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}, nil
}

// Send sends a Slack notification
func (c *SlackChannel) Send(notification *Notification) error {
	alert := notification.Alert

	color := "warning"
	if alert.Severity == SeverityCritical {
		color = "danger"
	} else if notification.IsResolution {
		color = "good"
	}

	attachment := map[string]interface{}{
		"color":     color,
		"title":     alert.RuleName,
		"text":      alert.Message,
		"fallback":  alert.Message,
		"timestamp": notification.Timestamp.Unix(),
		"fields": []map[string]interface{}{
			{
				"title": "Severity",
				"value": string(alert.Severity),
				"short": true,
			},
			{
				"title": "Value",
				"value": fmt.Sprintf("%.2f", alert.Value),
				"short": true,
			},
			{
				"title": "Threshold",
				"value": fmt.Sprintf("%.2f", alert.Threshold),
				"short": true,
			},
			{
				"title": "State",
				"value": string(alert.State),
				"short": true,
			},
		},
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{attachment},
	}

	if c.config.Channel != "" {
		payload["channel"] = c.config.Channel
	}
	if c.config.Username != "" {
		payload["username"] = c.config.Username
	}
	if c.config.IconEmoji != "" {
		payload["icon_emoji"] = c.config.IconEmoji
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(c.config.WebhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// GetType returns the channel type
func (c *SlackChannel) GetType() string {
	return "slack"
}

// IsEnabled returns whether the channel is enabled
func (c *SlackChannel) IsEnabled() bool {
	return c.config.Enabled
}

// ConsoleChannel implements console notifications
type ConsoleChannel struct {
	config ConsoleConfig
	logger *logrus.Logger
}

// ConsoleConfig holds console channel configuration
type ConsoleConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	Colored bool `json:"colored" yaml:"colored"`
}

// NewConsoleChannel creates a new console notification channel
func NewConsoleChannel(config map[string]interface{}, logger *logrus.Logger) (*ConsoleChannel, error) {
	data, _ := json.Marshal(config)
	var consoleConfig ConsoleConfig
	if err := json.Unmarshal(data, &consoleConfig); err != nil {
		consoleConfig = ConsoleConfig{Enabled: true, Colored: true}
	}

	return &ConsoleChannel{
		config: consoleConfig,
		logger: logger,
	}, nil
}

// Send sends a console notification
func (c *ConsoleChannel) Send(notification *Notification) error {
	alert := notification.Alert

	fields := logrus.Fields{
		"rule":      alert.RuleName,
		"severity":  alert.Severity,
		"value":     alert.Value,
		"threshold": alert.Threshold,
		"state":     alert.State,
	}

	for k, v := range alert.Labels {
		fields[k] = v
	}

	entry := c.logger.WithFields(fields)

	if notification.IsResolution {
		entry.Info("Alert resolved")
	} else {
		switch alert.Severity {
		case SeverityCritical:
			entry.Error("Alert triggered")
		case SeverityWarning:
			entry.Warn("Alert triggered")
		default:
			entry.Info("Alert triggered")
		}
	}

	return nil
}

// GetType returns the channel type
func (c *ConsoleChannel) GetType() string {
	return "console"
}

// IsEnabled returns whether the channel is enabled
func (c *ConsoleChannel) IsEnabled() bool {
	return c.config.Enabled
}