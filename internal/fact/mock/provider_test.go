package mock

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProvider(t *testing.T) {
	t.Run("Basic functionality", func(t *testing.T) {
		// Setup
		id := "test_fact"
		value := 42
		desc := "Test fact"
		provider := NewProvider(id, value, desc)

		// Check schema
		schema := provider.Describe()
		if schema.ID != id {
			t.Errorf("Expected schema ID %s, got %s", id, schema.ID)
		}
		if schema.Description != desc {
			t.Errorf("Expected schema description %s, got %s", desc, schema.Description)
		}

		// Get the fact
		fact, err := provider.Collect(context.Background(), "test-deployment", "test-stage")
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		// Check the fact properties
		if fact.ID() != id {
			t.Errorf("Expected fact ID %s, got %s", id, fact.ID())
		}
		if fact.Value() != value {
			t.Errorf("Expected fact value %v, got %v", value, fact.Value())
		}
		// Timestamp should be recent
		if time.Since(fact.Timestamp()) > time.Minute {
			t.Errorf("Expected recent timestamp, got: %v", fact.Timestamp())
		}
	})

	t.Run("WithError", func(t *testing.T) {
		// Setup
		expectedErr := errors.New("test error")
		provider := NewProvider("test_fact", 42, "Test fact").WithError(expectedErr)

		// Try to get the fact, expecting an error
		_, err := provider.Collect(context.Background(), "test-deployment", "test-stage")
		if err != expectedErr {
			t.Fatalf("Expected error %v but got: %v", expectedErr, err)
		}
	})

	t.Run("WithTimestamp", func(t *testing.T) {
		// Setup
		timestamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		provider := NewProvider("test_fact", 42, "Test fact").WithTimestamp(timestamp)

		// Get the fact
		fact, err := provider.Collect(context.Background(), "test-deployment", "test-stage")
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		// Check that the timestamp matches
		if !fact.Timestamp().Equal(timestamp) {
			t.Errorf("Expected timestamp %v, got %v", timestamp, fact.Timestamp())
		}
	})

	t.Run("Return appropriate gate.Fact", func(t *testing.T) {
		// Setup
		provider := NewProvider("test_fact", 42, "Test fact")

		// Get the fact
		result, err := provider.Collect(context.Background(), "test-deployment", "test-stage")
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		// Check that the return value implements gate.Fact
		_ = result // This is a compile-time check
	})
}
