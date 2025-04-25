package file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/asimihsan/planning_engine/internal/engine/opa"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

// Provider implements gate.PolicyProvider for file-based policy files
type Provider struct {
	PolicyPath string
	Query      string // e.g., "data.gate.response"
	// caches the loaded bundle to avoid reloading/recompiling every time
	cachedBundle gate.PolicyBundle
}

var _ gate.PolicyProvider = (*Provider)(nil)

// New creates a new file-based policy provider
func New(policyPath, query string) *Provider {
	return &Provider{
		PolicyPath: policyPath,
		Query:      query,
	}
}

// GetPolicyBundle implements gate.PolicyProvider
func (p *Provider) GetPolicyBundle(ctx context.Context) (gate.PolicyBundle, error) {
	// Basic caching to avoid reloading if already loaded
	if p.cachedBundle != nil {
		return p.cachedBundle, nil
	}

	// Read the policy file
	policyBytes, err := os.ReadFile(p.PolicyPath)
	if err != nil {
		return nil, fmt.Errorf("%w: reading policy file %s: %v", gate.ErrPolicyLoad, p.PolicyPath, err)
	}

	// Compile the policy module
	moduleName := filepath.Base(p.PolicyPath)
	compiler, err := ast.CompileModules(map[string]string{
		moduleName: string(policyBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: compiling policy module %s: %v", gate.ErrPolicyLoad, moduleName, err)
	}

	// Create and prepare the Rego query
	r := rego.New(
		rego.Query(p.Query),
		rego.Compiler(compiler),
	)

	// Prepare the query for evaluation
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: preparing policy query '%s': %v", gate.ErrPolicyLoad, p.Query, err)
	}

	// Calculate SHA256 of the policy file for versioning
	hash := sha256.Sum256(policyBytes)
	bundleID := hex.EncodeToString(hash[:])

	// Create and cache the policy bundle
	bundle := &opa.OpaPolicyBundle{
		BundleID:      bundleID,
		PreparedQuery: pq,
	}
	p.cachedBundle = bundle

	return bundle, nil
}
