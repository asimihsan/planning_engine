package levelsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/asimihsan/planning_engine/internal/metrics"
	"github.com/asimihsan/planning_engine/pkg/gate"
	"github.com/prometheus/client_golang/prometheus"
)

// Provider implements gate.FactProvider for a LevelServer API endpoint.
type Provider struct {
	baseURL     string
	factID      string
	httpClient  *http.Client
	cacheTTL    time.Duration
	description string

	mu          sync.RWMutex
	cachedValue gate.Fact
	expiry      time.Time
}

// NewProvider creates a new LevelServer fact provider.
func NewProvider(factID, baseURL string, cacheTTL time.Duration, description string) *Provider {
	return &Provider{
		baseURL:     baseURL,
		factID:      factID,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		cacheTTL:    cacheTTL,
		description: description,
	}
}

// Describe implements gate.FactProvider.
func (p *Provider) Describe() gate.Schema {
	return gate.Schema{
		ID:          p.factID,
		Description: p.description,
	}
}

// Collect implements gate.FactProvider.
func (p *Provider) Collect(ctx context.Context, deploymentID, stage string) (gate.Fact, error) {
	timer := prometheus.NewTimer(metrics.FactCollectLatency.WithLabelValues(p.factID))
	defer timer.ObserveDuration()

	// Check cache first
	p.mu.RLock()
	if p.cachedValue != nil && time.Now().Before(p.expiry) {
		cachedValue := p.cachedValue
		p.mu.RUnlock()
		return cachedValue, nil
	}
	p.mu.RUnlock()

	// Cache miss or expired, fetch fresh data
	url := fmt.Sprintf("%s/api/deployments/%s/stages/%s/metrics/%s",
		p.baseURL, deploymentID, stage, p.factID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		metrics.FactCollectErrors.WithLabelValues(p.factID, "request_creation").Inc()
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		metrics.FactCollectErrors.WithLabelValues(p.factID, "http_error").Inc()
		return nil, fmt.Errorf("%w: %v", gate.ErrFactSourceUnavailable, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			metrics.FactCollectErrors.WithLabelValues(p.factID, "body_close_error").Inc()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		metrics.FactCollectErrors.WithLabelValues(p.factID, fmt.Sprintf("status_%d", resp.StatusCode)).Inc()
		return nil, fmt.Errorf("%w: unexpected status code %d", gate.ErrFactSourceUnavailable, resp.StatusCode)
	}

	// Parse response
	var result struct {
		Value int `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		metrics.FactCollectErrors.WithLabelValues(p.factID, "decode_error").Inc()
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Create fact and update cache
	now := time.Now()
	fact := gate.NewFact(p.factID, result.Value, now)

	p.mu.Lock()
	p.cachedValue = fact
	p.expiry = now.Add(p.cacheTTL)
	p.mu.Unlock()

	return fact, nil
}
