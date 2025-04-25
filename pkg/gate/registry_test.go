package gate

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockFactProvider is an in-package mock for testing
type mockFactProvider struct {
	id    string
	desc  string
	value any
	err   error
}

func (m *mockFactProvider) Describe() Schema {
	return Schema{ID: m.id, Description: m.desc}
}

func (m *mockFactProvider) Collect(ctx context.Context, deploymentID, stage string) (Fact, error) {
	if m.err != nil {
		return nil, m.err
	}
	return NewFact(m.id, m.value, time.Now()), nil
}

func TestFactRegistry(t *testing.T) {
	t.Run("Register and GetProvider", func(t *testing.T) {
		// Setup
		registry := NewFactRegistry()
		provider := &mockFactProvider{id: "test_fact", desc: "Test fact", value: 42}

		// Register the provider
		registry.Register(provider)

		// Get the provider back
		retrieved, exists := registry.GetProvider("test_fact")
		if !exists {
			t.Fatalf("Expected provider to exist but it doesn't")
		}

		// Check that it's the same provider
		if retrieved != provider {
			t.Errorf("Retrieved provider is not the same as the registered one")
		}

		// Check that a non-existent provider doesn't exist
		_, exists = registry.GetProvider("nonexistent")
		if exists {
			t.Errorf("Expected non-existent provider to not exist but it does")
		}
	})

	t.Run("Snapshot successful", func(t *testing.T) {
		// Setup
		registry := NewFactRegistry()
		provider1 := &mockFactProvider{id: "fact1", desc: "Fact 1", value: 42}
		provider2 := &mockFactProvider{id: "fact2", desc: "Fact 2", value: "value"}

		// Register the providers
		registry.Register(provider1)
		registry.Register(provider2)

		// Get the snapshot
		facts, err := registry.Snapshot(context.Background(), "test-deployment", "test-stage")
		if err != nil {
			t.Fatalf("Expected no error but got: %v", err)
		}

		// Check the facts
		if facts["fact1"] != 42 {
			t.Errorf("Expected fact1 value to be 42, got: %v", facts["fact1"])
		}

		if facts["fact2"] != "value" {
			t.Errorf("Expected fact2 value to be 'value', got: %v", facts["fact2"])
		}
	})

	t.Run("Snapshot with error", func(t *testing.T) {
		// Setup
		registry := NewFactRegistry()
		expectedErr := errors.New("test error")
		provider := &mockFactProvider{id: "error_fact", desc: "Error fact", err: expectedErr}

		// Register the provider
		registry.Register(provider)

		// Get the snapshot, expecting an error
		_, err := registry.Snapshot(context.Background(), "test-deployment", "test-stage")
		if err == nil {
			t.Fatalf("Expected an error but got none")
		}

		// Check that the error message contains the fact ID and the original error
		if err.Error() != "collecting fact error_fact: test error" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}
