package integration

import (
	"context"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"

	"github.com/asimihsan/planning_engine/internal/audit/stdout"
	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/internal/engine/opa"
	configprovider "github.com/asimihsan/planning_engine/internal/fact/config"
	"github.com/asimihsan/planning_engine/internal/fact/levelsrv"
	"github.com/asimihsan/planning_engine/internal/fact/levelsrv_mock"
	"github.com/asimihsan/planning_engine/internal/policy/file"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

// TestLevelServerIntegration demonstrates the integration of the LevelServer provider
// with configuration-based fact providers, showing the complete end-to-end flow.
func TestLevelServerIntegration(t *testing.T) {
	// Create a context
	ctx := context.Background()

	// Start a mock LevelServer that mimics the real API
	mockServer := levelsrv_mock.NewServer()
	defer mockServer.Close()

	// Set the pending_delta value for our test deployment
	mockServer.SetPendingDelta("test-deployment", "test-stage", 300)

	// Create a test configuration
	testConfig := &config.AppConfig{
		FactProviders: &config.FactProviders{
			LevelServerBaseURL: mockServer.URL(),
			CacheTTL:           &pkl.Duration{Value: 5, Unit: pkl.Second},
			MaxStaleness:       &pkl.Duration{Value: 30, Unit: pkl.Second},
			ProviderTimeout:    &pkl.Duration{Value: 2, Unit: pkl.Second},
			MaxPendingAllowed:  500, // Allow up to 500 pending
		},
	}

	// Create the snapshot options
	snapshotOpts := gate.SnapshotOpts{
		MaxAge:             testConfig.FactProviders.MaxStaleness.GoDuration(),
		PerProviderTimeout: testConfig.FactProviders.ProviderTimeout.GoDuration(),
	}

	// Create other components
	registry := gate.NewFactRegistry()
	engine := opa.NewEngine()
	policyProvider := file.New("../../policy/rego/main.rego", "data.gate.response")
	logger := stdout.New()

	// Create and register fact providers
	// 1. LevelServer provider for pending_delta (dynamic value from external service)
	pendingDeltaProvider := levelsrv.NewProvider(
		"pending_delta",
		testConfig.FactProviders.LevelServerBaseURL,
		testConfig.FactProviders.CacheTTL.GoDuration(),
		"Number of devices newly targeted",
	)

	// 2. Config provider for max_pending_allowed (static value from configuration)
	maxPendingProvider := configprovider.NewMaxPendingAllowedProvider(testConfig)

	// Register both providers
	registry.Register(pendingDeltaProvider)
	registry.Register(maxPendingProvider)

	// Get the policy bundle
	policyBundle, err := policyProvider.GetPolicyBundle(ctx)
	if err != nil {
		t.Fatalf("Failed to get policy bundle: %v", err)
	}

	// Get the facts using the registry
	facts, err := registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", snapshotOpts)
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Verify that the facts were properly collected
	if facts["pending_delta"] != 300 {
		t.Errorf("Expected pending_delta=300, got: %v", facts["pending_delta"])
	}
	if facts["max_pending_allowed"] != 500 {
		t.Errorf("Expected max_pending_allowed=500, got: %v", facts["max_pending_allowed"])
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

	// Test a deny scenario by changing the LevelServer's pending_delta to exceed the max
	mockServer.SetPendingDelta("test-deployment", "test-stage", 600)

	// Clear the LevelServer provider's cache to force a fresh fetch
	// In a real application, the cache would naturally expire after the cacheTTL
	pendingDeltaProvider = levelsrv.NewProvider(
		"pending_delta",
		testConfig.FactProviders.LevelServerBaseURL,
		testConfig.FactProviders.CacheTTL.GoDuration(),
		"Number of devices newly targeted",
	)

	// Reset the registry and re-register providers
	registry = gate.NewFactRegistry()
	registry.Register(pendingDeltaProvider)
	registry.Register(maxPendingProvider)

	// Get the facts again
	facts, err = registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", snapshotOpts)
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Verify the facts were updated
	if facts["pending_delta"] != 600 {
		t.Errorf("Expected pending_delta=600, got: %v", facts["pending_delta"])
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

// TestConfigCachingBehavior demonstrates how configuration changes affect fact providers
func TestConfigCachingBehavior(t *testing.T) {
	// Create context
	ctx := context.Background()

	// Create initial config
	testConfig := &config.AppConfig{
		FactProviders: &config.FactProviders{
			MaxPendingAllowed: 500,
		},
	}

	// Create registry
	registry := gate.NewFactRegistry()

	// Create and register ConfigProvider
	maxPendingProvider := configprovider.NewMaxPendingAllowedProvider(testConfig)
	registry.Register(maxPendingProvider)

	// Get the initial fact value
	facts, err := registry.Snapshot(ctx, "test-deployment", "test-stage")
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Verify initial value
	if facts["max_pending_allowed"] != 500 {
		t.Errorf("Expected max_pending_allowed=500, got: %v", facts["max_pending_allowed"])
	}

	// Update the configuration
	testConfig.FactProviders.MaxPendingAllowed = 750

	// Get the facts again - should reflect the updated config
	facts, err = registry.Snapshot(ctx, "test-deployment", "test-stage")
	if err != nil {
		t.Fatalf("Failed to get facts: %v", err)
	}

	// Verify the updated value
	if facts["max_pending_allowed"] != 750 {
		t.Errorf("Expected max_pending_allowed=750, got: %v", facts["max_pending_allowed"])
	}
}
