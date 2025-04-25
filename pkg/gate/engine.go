package gate

import "context"

// PolicyEngine evaluates facts against a policy.
type PolicyEngine interface {
	// Evaluate runs the policy against the input facts using the provided bundle.
	// Returns the Decision on success.
	// Must return ErrPolicyEvaluation if the evaluation itself fails (distinct from fact/policy load errors).
	Evaluate(ctx context.Context, policy PolicyBundle, input map[string]any) (Decision, error)
}
