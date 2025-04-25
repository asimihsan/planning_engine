package gate

import (
	"testing"
	"time"
)

func TestBasicFact(t *testing.T) {
	// Setup
	id := "test_fact"
	value := 42
	now := time.Now()

	// Create a new fact
	fact := NewFact(id, value, now)

	// Verify the fact's properties
	if fact.ID() != id {
		t.Errorf("Expected ID %s, got %s", id, fact.ID())
	}

	if fact.Value() != value {
		t.Errorf("Expected value %v, got %v", value, fact.Value())
	}

	if !fact.Timestamp().Equal(now) {
		t.Errorf("Expected timestamp %v, got %v", now, fact.Timestamp())
	}
}
