package stdout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
)

func TestLogger(t *testing.T) {
	// These are primarily coverage tests, since we're just logging to stdout
	logger := New()
	ctx := context.Background()

	t.Run("LogDecision", func(t *testing.T) {
		input := map[string]any{
			"fact1": 42,
			"fact2": "value",
		}
		decision := gate.Decision{
			Allow:       true,
			DenyReasons: nil,
		}
		err := logger.LogDecision(ctx, input, decision, "test-policy", "test-config", time.Millisecond*50)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Test with a deny decision
		decision = gate.Decision{
			Allow:       false,
			DenyReasons: []string{"reason1", "reason2"},
		}
		err = logger.LogDecision(ctx, input, decision, "test-policy", "test-config", time.Millisecond*50)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("LogSystemError", func(t *testing.T) {
		err := logger.LogSystemError(ctx, errors.New("test error"), "test-deployment", "test-stage", "test-policy", "test-config")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Test with a standard error type
		err = logger.LogSystemError(ctx, gate.ErrFactStale, "test-deployment", "test-stage", "test-policy", "test-config")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})
}
