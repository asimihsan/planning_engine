package opa

import (
	"context"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
)

// Use a struct that doesn't implement gate.PolicyBundle correctly
type InvalidBundle struct{}

func (b InvalidBundle) ID() string { return "invalid" }

// Helper to create a test policy bundle
func createTestBundle(t *testing.T, policy string, query string) *OpaPolicyBundle {
	t.Helper()

	// Compile the policy
	compiler, err := ast.CompileModules(map[string]string{
		"test.rego": policy,
	})
	if err != nil {
		t.Fatalf("Failed to compile test policy: %v", err)
	}

	// Create a Rego object
	r := rego.New(
		rego.Query(query),
		rego.Compiler(compiler),
	)

	// Prepare for evaluation
	pq, err := r.PrepareForEval(context.Background())
	if err != nil {
		t.Fatalf("Failed to prepare query: %v", err)
	}

	return &OpaPolicyBundle{
		BundleID:      "test-bundle",
		PreparedQuery: pq,
	}
}

func TestEngine(t *testing.T) {
	// Simple policy for testing that follows our expected format
	policy := `
	package test
	
	default allow := false
	default deny_reasons := []
	
	allow if {
		input.value < 10
	}
	
	deny_reasons := ["value too high"] if {
		not allow
		input.value >= 10
	}
	
	response := {
		"allow": allow,
		"deny_reasons": deny_reasons
	} if true
	`

	t.Run("Allow decision", func(t *testing.T) {
		// Setup
		engine := NewEngine()
		bundle := createTestBundle(t, policy, "data.test.response")

		// Input that should result in an allow
		input := map[string]any{
			"value": 5,
		}

		// Evaluate
		decision, err := engine.Evaluate(context.Background(), bundle, input)
		if err != nil {
			t.Fatalf("Evaluation failed: %v", err)
		}

		// Verify
		if !decision.Allow {
			t.Errorf("Expected allow=true, got allow=false")
		}
		if len(decision.DenyReasons) != 0 {
			t.Errorf("Expected empty deny_reasons, got: %v", decision.DenyReasons)
		}
	})

	t.Run("Deny decision", func(t *testing.T) {
		// Setup
		engine := NewEngine()
		bundle := createTestBundle(t, policy, "data.test.response")

		// Input that should result in a deny
		input := map[string]any{
			"value": 15,
		}

		// Evaluate
		decision, err := engine.Evaluate(context.Background(), bundle, input)
		if err != nil {
			t.Fatalf("Evaluation failed: %v", err)
		}

		// Verify
		if decision.Allow {
			t.Errorf("Expected allow=false, got allow=true")
		}
		if len(decision.DenyReasons) != 1 || decision.DenyReasons[0] != "value too high" {
			t.Errorf("Expected deny_reasons=[\"value too high\"], got: %v", decision.DenyReasons)
		}
	})

	t.Run("Invalid policy bundle", func(t *testing.T) {
		// Setup
		engine := NewEngine()
		input := map[string]any{"value": 5}

		invalidBundle := InvalidBundle{}

		// Evaluate (should fail)
		_, err := engine.Evaluate(context.Background(), invalidBundle, input)
		if err == nil {
			t.Fatalf("Expected error for invalid bundle, got none")
		}
	})

	t.Run("Malformed policy result", func(t *testing.T) {
		// Define a policy that returns a non-standard format
		badPolicy := `
		package test
		
		response := "not a proper response object"
		`

		// Setup
		engine := NewEngine()
		bundle := createTestBundle(t, badPolicy, "data.test.response")
		input := map[string]any{"value": 5}

		// Evaluate (should fail with appropriate error)
		_, err := engine.Evaluate(context.Background(), bundle, input)
		if err == nil {
			t.Fatalf("Expected error for malformed policy result, got none")
		}
	})
}
