package opa

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/asimihsan/planning_engine/pkg/gate"
)

// OpaPolicyBundle is a concrete implementation of gate.PolicyBundle for OPA policies
type OpaPolicyBundle struct {
	BundleID      string
	PreparedQuery rego.PreparedEvalQuery
}

var _ gate.PolicyBundle = (*OpaPolicyBundle)(nil)

// ID implements gate.PolicyBundle
func (b *OpaPolicyBundle) ID() string {
	return b.BundleID
}

// Engine implements gate.PolicyEngine using OPA
type Engine struct{}

// NewEngine creates a new OPA policy engine
func NewEngine() *Engine {
	return &Engine{}
}

// Evaluate implements gate.PolicyEngine
func (e *Engine) Evaluate(ctx context.Context, policy gate.PolicyBundle, input map[string]any) (gate.Decision, error) {
	opaBundle, ok := policy.(*OpaPolicyBundle)
	if !ok {
		return gate.Decision{}, fmt.Errorf("%w: invalid policy bundle type: %T", gate.ErrPolicyEvaluation, policy)
	}

	resultSet, err := opaBundle.PreparedQuery.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return gate.Decision{}, fmt.Errorf("%w: evaluation failed: %v", gate.ErrPolicyEvaluation, err)
	}

	// Default deny if we can't interpret the results correctly
	decision := gate.Decision{Allow: false}

	// Parse the results based on our expected policy format
	// We expect the policy to define an "allow" boolean and "deny_reasons" array
	if len(resultSet) > 0 && len(resultSet[0].Expressions) > 0 {
		result, ok := resultSet[0].Expressions[0].Value.(map[string]interface{})
		if !ok {
			return gate.Decision{}, fmt.Errorf("%w: unexpected result format", gate.ErrPolicyEvaluation)
		}

		// Extract the allow flag
		if allow, ok := result["allow"].(bool); ok {
			decision.Allow = allow
		}

		// Extract deny reasons if present and the result is a denial
		if !decision.Allow {
			if reasons, ok := result["deny_reasons"].([]interface{}); ok {
				for _, r := range reasons {
					if reason, ok := r.(string); ok {
						decision.DenyReasons = append(decision.DenyReasons, reason)
					}
				}
			}
		}
	} else {
		return gate.Decision{}, fmt.Errorf("%w: policy result set is empty or malformed", gate.ErrPolicyEvaluation)
	}

	return decision, nil
}
