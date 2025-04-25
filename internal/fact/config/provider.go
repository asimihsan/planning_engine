// Package config provides the configuration-based fact provider
package config

import (
	"context"
	"time"

	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/internal/metrics"
	"github.com/asimihsan/planning_engine/pkg/gate"
	"github.com/prometheus/client_golang/prometheus"
)

// Provider implements gate.FactProvider for configuration-based facts.
type Provider struct {
	factID      string
	description string
	config      *config.AppConfig
	valueFunc   func(*config.AppConfig) any
}

// NewMaxPendingAllowedProvider creates a new provider for the max_pending_allowed fact.
func NewMaxPendingAllowedProvider(cfg *config.AppConfig) *Provider {
	return &Provider{
		factID:      "max_pending_allowed",
		description: "Maximum allowed devices in pending state",
		config:      cfg,
		valueFunc: func(cfg *config.AppConfig) any {
			return cfg.FactProviders.MaxPendingAllowed
		},
	}
}

// NewProvider creates a new configuration-based fact provider with a custom value function.
func NewProvider(factID, description string, config *config.AppConfig, valueFunc func(*config.AppConfig) any) *Provider {
	return &Provider{
		factID:      factID,
		description: description,
		config:      config,
		valueFunc:   valueFunc,
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
func (p *Provider) Collect(ctx context.Context, _, _ string) (gate.Fact, error) {
	timer := prometheus.NewTimer(metrics.FactCollectLatency.WithLabelValues(p.factID))
	defer timer.ObserveDuration()

	// Configuration facts are always fresh (current time)
	// and we don't need to make external calls
	value := p.valueFunc(p.config)
	return gate.NewFact(p.factID, value, time.Now()), nil
}
