package config

import (
	"context"
	"testing"
	"time"

	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigProvider(t *testing.T) {
	// Create a test AppConfig
	cfg := &config.AppConfig{
		FactProviders: &config.FactProviders{
			MaxPendingAllowed: 800,
		},
	}

	// Create our config provider
	provider := NewMaxPendingAllowedProvider(cfg)

	// Verify the provider description is correct
	schema := provider.Describe()
	assert.Equal(t, "max_pending_allowed", schema.ID)
	assert.Equal(t, "Maximum allowed devices in pending state", schema.Description)

	// Test that the provider returns the correct value
	fact, err := provider.Collect(context.Background(), "test-deployment", "test-stage")
	assert.NoError(t, err)
	assert.Equal(t, "max_pending_allowed", fact.ID())
	assert.Equal(t, 800, fact.Value())

	// Verify timestamp is recent
	assert.WithinDuration(t, time.Now(), fact.Timestamp(), time.Second)

	// Test that changing the config updates the provided value
	cfg.FactProviders.MaxPendingAllowed = 1000
	fact, err = provider.Collect(context.Background(), "test-deployment", "test-stage")
	assert.NoError(t, err)
	assert.Equal(t, 1000, fact.Value())
}

func TestCustomConfigProvider(t *testing.T) {
	// Create a test AppConfig
	cfg := &config.AppConfig{
		FactProviders: &config.FactProviders{
			MaxPendingAllowed: 500,
		},
		Prometheus: &config.Prometheus{
			ListenAddr: ":9100",
		},
	}

	// Create a custom provider that extracts a different config value
	provider := NewProvider(
		"prometheus_port",
		"Prometheus metrics port",
		cfg,
		func(c *config.AppConfig) any {
			return c.Prometheus.ListenAddr
		},
	)

	// Verify schema
	schema := provider.Describe()
	assert.Equal(t, "prometheus_port", schema.ID)
	assert.Equal(t, "Prometheus metrics port", schema.Description)

	// Test value extraction
	fact, err := provider.Collect(context.Background(), "", "")
	assert.NoError(t, err)
	assert.Equal(t, ":9100", fact.Value())
}
