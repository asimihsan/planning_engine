package gate

import (
	"context"
	"time"
)

// Fact represents a single piece of data about the system state, with timestamp.
type Fact interface {
	ID() string           // e.g., "pending_delta"
	Value() any           // The actual data point
	Timestamp() time.Time // When the fact data was considered current
}

// Schema provides metadata about a Fact or input structure.
type Schema struct {
	ID          string
	Description string
}

// FactProvider fetches or calculates a specific Fact.
type FactProvider interface {
	Describe() Schema
	// Collect fetches the fact. Implementations handle caching & staleness checks.
	// Must return ErrFactStale or ErrFactSourceUnavailable for critical failures.
	Collect(ctx context.Context, deploymentID, stage string) (Fact, error)
}

// BasicFact is a concrete implementation of the Fact interface
type BasicFact struct {
	FactID    string
	FactValue any
	FactTime  time.Time
}

func (f BasicFact) ID() string           { return f.FactID }
func (f BasicFact) Value() any           { return f.FactValue }
func (f BasicFact) Timestamp() time.Time { return f.FactTime }

// NewFact creates a new Fact with the given ID, value, and timestamp
func NewFact(id string, value any, timestamp time.Time) Fact {
	return BasicFact{
		FactID:    id,
		FactValue: value,
		FactTime:  timestamp,
	}
}
