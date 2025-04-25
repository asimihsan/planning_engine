package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
	"github.com/open-policy-agent/opa/v1/rego"
)

// Engine implements the PolicyEngine interface using OPA.
type Engine struct {
	// Compiled OPA query for policy evaluation
	query rego.PreparedEvalQuery
	// SHA of the policy bundle - for audit and tracking
	policySHA string
}

// NewEngine creates a new OPA-based policy engine.
func NewEngine(policy gate.PolicyBundle) (*Engine, error) {
	// Compile the OPA query
	query, err := rego.New(
		rego.Query("data.gate.response"),
		rego.Module("policy.rego", string(policy.Data())),
	).PrepareForEval(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", gate.ErrPolicyEvaluation, err)
	}

	return &Engine{
		query:     query,
		policySHA: policy.ID(),
	}, nil
}

// Evaluate implements the PolicyEngine interface.
func (e *Engine) Evaluate(ctx context.Context, input map[string]any) (gate.Decision, error) {
	startTime := time.Now()

	// Execute the OPA query
	results, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return gate.Decision{}, fmt.Errorf("%w: %v", gate.ErrPolicyEvaluation, err)
	}

	// Check if we have any results
	if len(results) == 0 || len(results[0].Expressions) == 0 {
		return gate.Decision{}, fmt.Errorf("%w: no results from policy evaluation", gate.ErrPolicyEvaluation)
	}

	// Extract the response from the result
	responseValue := results[0].Expressions[0].Value
	responseMap, ok := responseValue.(map[string]interface{})
	if !ok {
		return gate.Decision{}, fmt.Errorf("%w: unexpected response format", gate.ErrPolicyEvaluation)
	}

	// Create and return the decision
	decision := gate.Decision{
		Allow:        responseMap["allow"].(bool),
		DenyReasons:  extractStringArray(responseMap["deny_reasons"]),
		PolicySHA:    e.policySHA,
		EvalDuration: time.Since(startTime),
	}

	return decision, nil
}

// Helper to extract a string array from an interface{}
func extractStringArray(value interface{}) []string {
	if value == nil {
		return nil
	}

	// Try to extract as []interface{} first (common OPA return pattern)
	if arr, ok := value.([]interface{}); ok {
		result := make([]string, len(arr))
		for i, v := range arr {
			if str, ok := v.(string); ok {
				result[i] = str
			} else {
				// Try to convert to JSON string
				b, err := json.Marshal(v)
				if err == nil {
					result[i] = string(b)
				} else {
					result[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		return result
	}

	// Try as direct []string
	if arr, ok := value.([]string); ok {
		return arr
	}

	// Return a single item array if it's a string
	if str, ok := value.(string); ok {
		return []string{str}
	}

	// Return empty array as fallback
	return []string{}
}
