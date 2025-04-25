package gate

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// SnapshotOpts lets the caller tune latency / staleness guarantees.
type SnapshotOpts struct {
	MaxAge             time.Duration // zero => no age check
	PerProviderTimeout time.Duration // enforced with ctx.WithTimeout
}

// FactRegistry holds a collection of FactProviders and orchestrates fact collection.
type FactRegistry struct {
	providers map[string]FactProvider
	mu        sync.RWMutex
}

// NewFactRegistry creates a new empty FactRegistry.
func NewFactRegistry() *FactRegistry {
	return &FactRegistry{
		providers: make(map[string]FactProvider),
	}
}

// Register adds a FactProvider to the registry.
// If a provider with the same ID already exists, it will be replaced.
func (r *FactRegistry) Register(provider FactProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	schema := provider.Describe()
	r.providers[schema.ID] = provider
}

// GetProvider retrieves a FactProvider by ID.
func (r *FactRegistry) GetProvider(factID string) (FactProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[factID]
	return provider, exists
}

// Snapshot collects all facts from registered providers for the given deployment and stage.
// Returns a map of fact ID to fact value, suitable for policy evaluation.
// For backward compatibility with existing code that doesn't specify options.
func (r *FactRegistry) Snapshot(ctx context.Context, deploymentID, stage string) (map[string]any, error) {
	return r.SnapshotWithOpts(ctx, deploymentID, stage, SnapshotOpts{})
}

// SnapshotWithOpts collects all facts from registered providers with the given options.
// Uses parallel collection with errgroup and applies staleness checks.
func (r *FactRegistry) SnapshotWithOpts(ctx context.Context, deploymentID, stage string, opts SnapshotOpts) (map[string]any, error) {
	r.mu.RLock()
	// Create a copy of the providers map to avoid holding the lock during collection
	providers := make(map[string]FactProvider, len(r.providers))
	for id, provider := range r.providers {
		providers[id] = provider
	}
	r.mu.RUnlock()

	// Set up errgroup for parallel collection
	g, gctx := errgroup.WithContext(ctx)

	// Channel to collect results from goroutines
	type result struct {
		id  string
		val any
		err error
	}
	results := make(chan result, len(providers))

	// Launch a goroutine for each provider
	for id, provider := range providers {
		id, provider := id, provider // Capture loop variables
		g.Go(func() error {
			// Apply per-provider timeout if specified
			pctx := gctx
			if opts.PerProviderTimeout > 0 {
				var cancel context.CancelFunc
				pctx, cancel = context.WithTimeout(gctx, opts.PerProviderTimeout)
				defer cancel()
			}

			// Collect the fact
			fact, err := provider.Collect(pctx, deploymentID, stage)
			if err != nil {
				results <- result{id: id, err: fmt.Errorf("collecting fact %s: %w", id, err)}
				return nil // We collect errors via channel, don't fail the errgroup
			}

			// Check staleness if MaxAge is specified
			if opts.MaxAge > 0 && time.Since(fact.Timestamp()) > opts.MaxAge {
				results <- result{id: id, err: fmt.Errorf("collecting fact %s: %w", id, ErrFactStale)}
				return nil
			}

			// Send successful result
			results <- result{id: fact.ID(), val: fact.Value(), err: nil}
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, err // This shouldn't happen as we collect errors via channel
	}
	close(results)

	// Process results
	resultMap := make(map[string]any, len(providers))
	var firstErr error

	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		resultMap[res.id] = res.val
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return resultMap, nil
}
