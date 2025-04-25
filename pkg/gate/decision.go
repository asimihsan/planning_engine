package gate

import (
	"time"
)

// Decision represents the outcome of a policy evaluation.
type Decision struct {
	Allow        bool          // Whether the operation is allowed
	DenyReasons  []string      // Machine-readable explanations if Allow is false
	PolicySHA    string        // Identifier for the policy version used
	ConfigSHA    string        // Identifier for the configuration version used
	EvalDuration time.Duration // How long the evaluation took
}
