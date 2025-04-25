package integration

import (
	"context"
	"testing"
	"time"

	"github.com/asimihsan/planning_engine/internal/audit/stdout"
	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/internal/engine/opa"
	"github.com/asimihsan/planning_engine/internal/fact/mock"
	"github.com/asimihsan/planning_engine/internal/policy/file"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

func TestBasicIntegration(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadFromPath(ctx, "../../policy/local/local.pkl")
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Create the snapshot options
	snapshotOpts := gate.SnapshotOpts{
		MaxAge:             cfg.FactProviders.MaxStaleness.GoDuration(),
		PerProviderTimeout: cfg.FactProviders.ProviderTimeout.GoDuration(),
	}

	// Create other components
	registry := gate.NewFactRegistry()
	engine := opa.NewEngine()
	policyProvider := file.New("../../policy/rego/main.rego", "data.gate.response")
	logger := stdout.New()

	// Register mock fact providers
	pendingDeltaProvider := mock.NewProvider("pending_delta", 100, "Number of devices newly targeted")
	maxPendingProvider := mock.NewProvider("max_pending_allowed", 500, "Maximum allowed devices in pending state")
	registry.Register(pendingDeltaProvider)
	registry.Register(maxPendingProvider)

	// Get the policy bundle
	policyBundle, err := policyProvider.GetPolicyBundle(ctx)
	if err != nil {
		t.Fatalf("Failed to get policy bundle: %v", err)
	}

	// Get the facts using the new SnapshotWithOpts method
	facts, err := registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", snapshotOpts)
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

	// Get the facts again using the options
	facts, err = registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", snapshotOpts)
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

// TestStalenessBehavior verifies that the registry correctly enforces the fact staleness policy
func TestStalenessBehavior(t *testing.T) {
	// Create context
	ctx := context.Background()

	// Create registry
	registry := gate.NewFactRegistry()

	// Create a fact provider with a "stale" timestamp (10 minutes old)
	staleTime := time.Now().Add(-10 * time.Minute)
	staleProvider := mock.NewProviderWithTimestamp("test_fact", 123, "Test fact", staleTime)
	registry.Register(staleProvider)

	// Try to collect with a 5 minute staleness threshold
	opts := gate.SnapshotOpts{
		MaxAge: 5 * time.Minute,
	}

	// Should fail with staleness error
	_, err := registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", opts)
	if err == nil {
		t.Fatal("Expected error for stale fact, got nil")
	}

	if !gate.IsWrappingError(err, gate.ErrFactStale) {
		t.Errorf("Expected ErrFactStale, got: %v", err)
	}

	// Create a fresh fact provider
	freshProvider := mock.NewProvider("test_fact", 123, "Test fact")
	registry = gate.NewFactRegistry()
	registry.Register(freshProvider)

	// Try again - should work now
	facts, err := registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", opts)
	if err != nil {
		t.Fatalf("Unexpected error with fresh fact: %v", err)
	}

	if facts["test_fact"] != 123 {
		t.Errorf("Expected test_fact=123, got: %v", facts["test_fact"])
	}
}
