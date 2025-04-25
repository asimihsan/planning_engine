package decision

import (
	"context"
	"testing"

	"github.com/asimihsan/planning_engine/pkg/gate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testPolicyBundle struct {
	id   string
	data []byte
}

func (b testPolicyBundle) ID() string   { return b.id }
func (b testPolicyBundle) Data() []byte { return b.data }

func TestEngine_Evaluate(t *testing.T) {
	// Create a test policy bundle with a simple rule
	policyData := []byte(`
package gate

default allow := false
default deny_reasons := []

# Simple policy for testing: check if pending_delta is within allowed limit
allow if {
    input.pending_delta <= input.max_pending_allowed
}

deny_reasons := ["pending_delta exceeds allowed limit"] if {
    not allow
    input.pending_delta > input.max_pending_allowed
}

# Return a structured response for easier consumption by the engine
response := {
    "allow": allow,
    "deny_reasons": deny_reasons
} if true
`)
	bundle := testPolicyBundle{
		id:   "test-policy-sha",
		data: policyData,
	}

	// Create engine with the test policy
	engine, err := NewEngine(bundle)
	require.NoError(t, err)
	require.NotNil(t, engine)

	tests := []struct {
		name        string
		input       map[string]interface{}
		wantAllow   bool
		wantReasons []string
	}{
		{
			name: "Allow - within limits",
			input: map[string]interface{}{
				"pending_delta":       100,
				"max_pending_allowed": 500,
			},
			wantAllow:   true,
			wantReasons: []string{},
		},
		{
			name: "Deny - exceeds limit",
			input: map[string]interface{}{
				"pending_delta":       600,
				"max_pending_allowed": 500,
			},
			wantAllow:   false,
			wantReasons: []string{"pending_delta exceeds allowed limit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := engine.Evaluate(context.Background(), tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.wantAllow, decision.Allow)
			assert.Equal(t, tt.wantReasons, decision.DenyReasons)
			assert.Equal(t, "test-policy-sha", decision.PolicySHA)
			assert.True(t, decision.EvalDuration > 0)
		})
	}
}

func TestEngine_EvaluateInvalidInput(t *testing.T) {
	// Create a test policy bundle that requires specific inputs
	policyData := []byte(`
package gate

default allow := false

allow if {
    input.required_field > 0
}

response := {
    "allow": allow,
    "deny_reasons": []
} if true
`)
	bundle := testPolicyBundle{
		id:   "test-policy-sha",
		data: policyData,
	}

	// Create engine with the test policy
	engine, err := NewEngine(bundle)
	require.NoError(t, err)

	// Test with missing required field
	input := map[string]interface{}{
		"some_other_field": 42,
	}

	// Evaluation should succeed but return false since the rule condition isn't met
	decision, err := engine.Evaluate(context.Background(), input)
	require.NoError(t, err)
	assert.False(t, decision.Allow)
}

func TestEngine_InvalidPolicy(t *testing.T) {
	// Test with syntactically invalid Rego
	badPolicyData := []byte(`
package gate

default allow := false

# Syntax error - missing closing brace
allow if {
    input.value > 10
    input.other < 20
`)
	badBundle := testPolicyBundle{
		id:   "bad-policy-sha",
		data: badPolicyData,
	}

	// Creating engine should fail with syntax error
	_, err := NewEngine(badBundle)
	assert.Error(t, err)
	assert.True(t, gate.IsWrappingError(err, gate.ErrPolicyEvaluation))
}
