package gate

import "errors"

// Standard error types for gate operations
var (
	ErrFactSourceUnavailable = errors.New("gate: fact source unavailable")
	ErrFactStale             = errors.New("gate: fact data is stale")
	ErrPolicyEvaluation      = errors.New("gate: policy evaluation failed")
	ErrPolicyLoad            = errors.New("gate: policy bundle could not be loaded")
	ErrConfigLoad            = errors.New("gate: configuration could not be loaded")
)

// IsWrappingError checks if err is wrapping the target error using errors.Is.
// This is a helper for testing error wrapping.
func IsWrappingError(err, target error) bool {
	return errors.Is(err, target)
}
