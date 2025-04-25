package integration

import (
	"context"
	"testing"
	"time"

	"github.com/asimihsan/planning_engine/internal/audit/stdout"
	"github.com/asimihsan/planning_engine/internal/engine/opa"
	"github.com/asimihsan/planning_engine/internal/fact/mock"
	"github.com/asimihsan/planning_engine/internal/policy/file"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

func TestBasicIntegration(t *testing.T) {
	// Create the components
	registry := gate.NewFactRegistry()
	engine := opa.NewEngine()
	policyProvider := file.New("../../policy/rego/main.rego", "data.gate.response")
	logger := stdout.New()

	// Register mock fact providers
	pendingDeltaProvider := mock.NewProvider("pending_delta", 100, "Number of devices newly targeted")
	maxPendingProvider := mock.NewProvider("max_pending_allowed", 500, "Maximum allowed devices in pending state")
	registry.Register(pendingDeltaProvider)
	registry.Register(maxPendingProvider)

	// Create a context
	ctx := context.Background()

	// Get the policy bundle
	policyBundle, err := policyProvider.GetPolicyBundle(ctx)
	if err != nil {
		t.Fatalf("Failed to get policy bundle: %v", err)
	}

	// Get the facts
	facts, err := registry.Snapshot(ctx, "test-deployment", "test-stage")
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Evaluate the policy
	start := time.Now()
	decision, err := engine.Evaluate(ctx, policyBundle, facts)
	evalDuration := time.Since(start)
	if err != nil {
		t.Fatalf("Failed to evaluate policy: %v", err)
	}

	// Log the decision
	err = logger.LogDecision(ctx, facts, decision, policyBundle.ID(), "test-config", evalDuration)
	if err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// Verify the decision (allow since pending_delta < max_pending_allowed)
	if !decision.Allow {
		t.Errorf("Expected decision to be allow, got deny with reasons: %v", decision.DenyReasons)
	}

	// Test a deny scenario by modifying the pending_delta
	pendingDeltaProvider = mock.NewProvider("pending_delta", 600, "Number of devices newly targeted")
	maxPendingProvider = mock.NewProvider("max_pending_allowed", 500, "Maximum allowed devices in pending state")
	registry = gate.NewFactRegistry() // Reset the registry
	registry.Register(pendingDeltaProvider)
	registry.Register(maxPendingProvider)

	// Get the facts again
	facts, err = registry.Snapshot(ctx, "test-deployment", "test-stage")
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Evaluate the policy again
	start = time.Now()
	decision, err = engine.Evaluate(ctx, policyBundle, facts)
	evalDuration = time.Since(start)
	if err != nil {
		t.Fatalf("Failed to evaluate policy: %v", err)
	}

	// Log the decision
	err = logger.LogDecision(ctx, facts, decision, policyBundle.ID(), "test-config", evalDuration)
	if err != nil {
		t.Fatalf("Failed to log decision: %v", err)
	}

	// Verify the decision (deny since pending_delta > max_pending_allowed)
	if decision.Allow {
		t.Errorf("Expected decision to be deny, got allow")
	}
	if len(decision.DenyReasons) != 1 || decision.DenyReasons[0] != "pending_delta exceeds allowed limit" {
		t.Errorf("Expected deny reason 'pending_delta exceeds allowed limit', got: %v", decision.DenyReasons)
	}
}
