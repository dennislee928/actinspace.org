package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WebhookConfig represents configuration for a webhook endpoint
type WebhookConfig struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Method      string            `json:"method"` // POST, PUT, etc.
	Headers     map[string]string `json:"headers"`
	Enabled     bool              `json:"enabled"`
	EventTypes  []string          `json:"event_types"` // Filter by event types
	RetryCount  int               `json:"retry_count"`
	TimeoutSecs int               `json:"timeout_secs"`
}

// WebhookManager manages webhook integrations
type WebhookManager struct {
	mu       sync.RWMutex
	webhooks map[string]*WebhookConfig
	client   *http.Client
	queue    chan WebhookDelivery
	workers  int
}

// WebhookDelivery represents a webhook delivery attempt
type WebhookDelivery struct {
	Config    *WebhookConfig
	Payload   interface{}
	Timestamp time.Time
	Attempt   int
}

// WebhookResult represents the result of a webhook delivery
type WebhookResult struct {
	Success    bool      `json:"success"`
	StatusCode int       `json:"status_code"`
	Error      string    `json:"error,omitempty"`
	Duration   float64   `json:"duration_ms"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(workers int) *WebhookManager {
	manager := &WebhookManager{
		webhooks: make(map[string]*WebhookConfig),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		queue:   make(chan WebhookDelivery, 1000),
		workers: workers,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go manager.worker()
	}

	return manager
}

// RegisterWebhook registers a new webhook endpoint
func (m *WebhookManager) RegisterWebhook(config WebhookConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("webhook name is required")
	}
	if config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}
	if config.Method == "" {
		config.Method = "POST"
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.TimeoutSecs == 0 {
		config.TimeoutSecs = 10
	}

	m.webhooks[config.Name] = &config
	return nil
}

// UnregisterWebhook removes a webhook endpoint
func (m *WebhookManager) UnregisterWebhook(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.webhooks, name)
}

// SendEvent sends an event to all registered webhooks
func (m *WebhookManager) SendEvent(eventType string, payload interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, config := range m.webhooks {
		if !config.Enabled {
			continue
		}

		// Filter by event type if specified
		if len(config.EventTypes) > 0 {
			matched := false
			for _, et := range config.EventTypes {
				if et == eventType || et == "*" {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Queue delivery
		delivery := WebhookDelivery{
			Config:    config,
			Payload:   payload,
			Timestamp: time.Now(),
			Attempt:   0,
		}

		select {
		case m.queue <- delivery:
			// Queued successfully
		default:
			// Queue full, log error
			fmt.Printf("Webhook queue full, dropping event for %s\n", config.Name)
		}
	}
}

// worker processes webhook deliveries from the queue
func (m *WebhookManager) worker() {
	for delivery := range m.queue {
		result := m.deliver(delivery)

		// Retry on failure
		if !result.Success && delivery.Attempt < delivery.Config.RetryCount {
			delivery.Attempt++
			// Exponential backoff
			backoff := time.Duration(1<<uint(delivery.Attempt)) * time.Second
			time.Sleep(backoff)

			select {
			case m.queue <- delivery:
				// Requeued for retry
			default:
				fmt.Printf("Failed to requeue webhook delivery for %s\n", delivery.Config.Name)
			}
		}
	}
}

// deliver performs the actual HTTP request to the webhook endpoint
func (m *WebhookManager) deliver(delivery WebhookDelivery) WebhookResult {
	start := time.Now()
	result := WebhookResult{
		Timestamp: start,
	}

	// Prepare payload
	payloadBytes, err := json.Marshal(delivery.Payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		return result
	}

	// Create request
	req, err := http.NewRequest(delivery.Config.Method, delivery.Config.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Space-SOC-Webhook/1.0")
	for key, value := range delivery.Config.Headers {
		req.Header.Set(key, value)
	}

	// Set timeout
	client := &http.Client{
		Timeout: time.Duration(delivery.Config.TimeoutSecs) * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.Duration = time.Since(start).Seconds() * 1000
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start).Seconds() * 1000

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		result.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
	}

	return result
}

// GetWebhooks returns all registered webhooks
func (m *WebhookManager) GetWebhooks() map[string]*WebhookConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	webhooks := make(map[string]*WebhookConfig)
	for name, config := range m.webhooks {
		webhooks[name] = config
	}
	return webhooks
}

// GetQueueSize returns the current queue size
func (m *WebhookManager) GetQueueSize() int {
	return len(m.queue)
}

