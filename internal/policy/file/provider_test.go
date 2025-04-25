package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asimihsan/planning_engine/internal/engine/opa"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

func TestProvider(t *testing.T) {
	// Create a temporary policy file
	policyContent := `
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

	tmpDir := t.TempDir()
	policyFile := filepath.Join(tmpDir, "test_policy.rego")
	if err := os.WriteFile(policyFile, []byte(policyContent), 0o644); err != nil {
		t.Fatalf("Failed to create test policy file: %v", err)
	}

	t.Run("Load policy successfully", func(t *testing.T) {
		// Setup
		provider := New(policyFile, "data.test.response")

		// Get the policy bundle
		bundle, err := provider.GetPolicyBundle(context.Background())
		if err != nil {
			t.Fatalf("Failed to get policy bundle: %v", err)
		}

		// Verify the bundle
		if bundle == nil {
			t.Fatalf("Expected non-nil bundle, got nil")
		}

		// Check that bundle has a non-empty ID
		if bundle.ID() == "" {
			t.Errorf("Expected non-empty bundle ID, got empty string")
		}

		// Verify that it's an OpaPolicyBundle
		_, ok := bundle.(*opa.OpaPolicyBundle)
		if !ok {
			t.Errorf("Expected *opa.OpaPolicyBundle, got %T", bundle)
		}
	})

	t.Run("Cache loaded policy", func(t *testing.T) {
		// Setup
		provider := New(policyFile, "data.test.response")

		// Get the policy bundle twice
		bundle1, err := provider.GetPolicyBundle(context.Background())
		if err != nil {
			t.Fatalf("Failed to get policy bundle (first call): %v", err)
		}

		bundle2, err := provider.GetPolicyBundle(context.Background())
		if err != nil {
			t.Fatalf("Failed to get policy bundle (second call): %v", err)
		}

		// Verify that the bundles are the same instance (cached)
		if bundle1 != bundle2 {
			t.Errorf("Expected cached bundle to be returned, got different instances")
		}
	})

	t.Run("Handle nonexistent file", func(t *testing.T) {
		// Setup
		nonexistentFile := filepath.Join(tmpDir, "nonexistent.rego")
		provider := New(nonexistentFile, "data.test.response")

		// Try to get the policy bundle, expecting an error
		_, err := provider.GetPolicyBundle(context.Background())
		if err == nil {
			t.Fatalf("Expected error for nonexistent file, got none")
		}

		// Verify that the error is wrapped with ErrPolicyLoad
		if !gate.IsWrappingError(err, gate.ErrPolicyLoad) {
			t.Errorf("Expected error to wrap ErrPolicyLoad, got: %v", err)
		}
	})

	t.Run("Handle invalid Rego syntax", func(t *testing.T) {
		// Create a file with invalid Rego syntax
		invalidContent := `
		package test
		
		this is not valid Rego syntax
		`
		invalidFile := filepath.Join(tmpDir, "invalid.rego")
		if err := os.WriteFile(invalidFile, []byte(invalidContent), 0o644); err != nil {
			t.Fatalf("Failed to create invalid policy file: %v", err)
		}

		// Setup
		provider := New(invalidFile, "data.test.response")

		// Try to get the policy bundle, expecting an error
		_, err := provider.GetPolicyBundle(context.Background())
		if err == nil {
			t.Fatalf("Expected error for invalid Rego syntax, got none")
		}

		// Verify that the error is wrapped with ErrPolicyLoad
		if !gate.IsWrappingError(err, gate.ErrPolicyLoad) {
			t.Errorf("Expected error to wrap ErrPolicyLoad, got: %v", err)
		}
	})
}
