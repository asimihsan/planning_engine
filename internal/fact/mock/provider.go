package mock

import (
	"context"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
)

// Provider implements gate.FactProvider with controllable mock values.
type Provider struct {
	FactID      string
	Value       any
	Timestamp   time.Time
	Err         error
	Description string
}

var _ gate.FactProvider = (*Provider)(nil)

// NewProvider creates a new mock provider with the given ID and value.
func NewProvider(id string, value any, description string) *Provider {
	return &Provider{
		FactID:      id,
		Value:       value,
		Timestamp:   time.Now(),
		Description: description,
	}
}

// NewProviderWithTimestamp creates a new mock provider with a specific timestamp.
func NewProviderWithTimestamp(id string, value any, description string, timestamp time.Time) *Provider {
	return &Provider{
		FactID:      id,
		Value:       value,
		Timestamp:   timestamp,
		Description: description,
	}
}

// WithError configures the provider to return the specified error.
func (p *Provider) WithError(err error) *Provider {
	p.Err = err
	return p
}

// WithTimestamp sets a specific timestamp for the fact.
func (p *Provider) WithTimestamp(t time.Time) *Provider {
	p.Timestamp = t
	return p
}

// Describe implements gate.FactProvider.
func (p *Provider) Describe() gate.Schema {
	return gate.Schema{
		ID:          p.FactID,
		Description: p.Description,
	}
}

// Collect implements gate.FactProvider.
func (p *Provider) Collect(ctx context.Context, deploymentID, stage string) (gate.Fact, error) {
	if p.Err != nil {
		return nil, p.Err
	}

	return gate.NewFact(p.FactID, p.Value, p.Timestamp), nil
}
