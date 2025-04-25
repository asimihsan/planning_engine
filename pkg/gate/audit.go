package gate

import (
	"context"
	"time"
)

// AuditLogger persists decision and error information.
type AuditLogger interface {
	// LogDecision records the outcome of a successful evaluation.
	// input: the facts map given to Evaluate.
	// decision: the Decision struct returned by Evaluate.
	// policyID, configID: identifiers for traceability.
	// evalDuration: time taken for the Evaluate call.
	LogDecision(ctx context.Context, input map[string]any, decision Decision, policyID, configID string, evalDuration time.Duration) error

	// LogSystemError records failures occurring outside successful policy evaluation.
	// systemError: The specific error (e.g., ErrFactStale, ErrPolicyLoad).
	// deploymentID, stage: Context for the operation attempt.
	// policyID, configID: Identifiers if available at the time of error.
	LogSystemError(ctx context.Context, systemError error, deploymentID, stage, policyID, configID string) error
}
