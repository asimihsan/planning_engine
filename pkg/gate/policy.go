package gate

import "context"

// PolicyBundle holds the compiled policy and metadata.
type PolicyBundle interface {
	ID() string   // e.g., SHA or version of the bundle content
	Data() []byte // The policy data
}

// PolicyProvider retrieves PolicyBundles.
type PolicyProvider interface {
	// GetPolicyBundle fetches the current policy bundle (e.g., from file/S3).
	// Implementations handle polling/updates. Should return ErrPolicyLoad on failure.
	GetPolicyBundle(ctx context.Context) (PolicyBundle, error)
}
