package gate

import (
	"context"
	"fmt"
	"sync"
)

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
func (r *FactRegistry) Snapshot(ctx context.Context, deploymentID, stage string) (map[string]any, error) {
	r.mu.RLock()
	// Create a copy of the providers map to avoid holding the lock during collection
	providers := make(map[string]FactProvider, len(r.providers))
	for id, provider := range r.providers {
		providers[id] = provider
	}
	r.mu.RUnlock()

	result := make(map[string]any)

	// For now, collect facts sequentially. In a later milestone, we'll use errgroup for parallelism.
	for id, provider := range providers {
		fact, err := provider.Collect(ctx, deploymentID, stage)
		if err != nil {
			return nil, fmt.Errorf("collecting fact %s: %w", id, err)
		}

		result[fact.ID()] = fact.Value()
	}

	return result, nil
}
