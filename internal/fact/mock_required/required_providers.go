// Package mock_required provides the required fact providers for static check
package mock_required

import (
	"context"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
)

// PendingDeltaProvider provides the pending_delta fact
type PendingDeltaProvider struct{}

var _ gate.FactProvider = (*PendingDeltaProvider)(nil)

// Describe implements the FactProvider interface
func (p *PendingDeltaProvider) Describe() gate.Schema {
	return gate.Schema{
		ID:          "pending_delta",
		Description: "Number of devices newly targeted",
	}
}

// Collect implements the FactProvider interface
func (p *PendingDeltaProvider) Collect(_ context.Context, _, _ string) (gate.Fact, error) {
	return gate.NewFact("pending_delta", 0, time.Now()), nil
}

// MaxPendingProvider provides the max_pending_allowed fact
type MaxPendingProvider struct{}

var _ gate.FactProvider = (*MaxPendingProvider)(nil)

// Describe implements the FactProvider interface
func (p *MaxPendingProvider) Describe() gate.Schema {
	return gate.Schema{
		ID:          "max_pending_allowed",
		Description: "Maximum allowed devices in pending state",
	}
}

// Collect implements the FactProvider interface
func (p *MaxPendingProvider) Collect(_ context.Context, _, _ string) (gate.Fact, error) {
	return gate.NewFact("max_pending_allowed", 0, time.Now()), nil
}
