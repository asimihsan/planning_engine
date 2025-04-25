package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

// Snapshot represents a cached configuration with metadata
type snapshot struct {
	cfg   *config.AppConfig
	sha   string    // SHA-256 hash of the file content
	mtime time.Time // Last modification time
}

// Cached configuration for atomic access
var cachedConfig atomic.Value // *snapshot

// LoadFromPathWithSHA loads and caches a PKL configuration file and returns the config along with its SHA
func LoadFromPathWithSHA(ctx context.Context, path string) (*config.AppConfig, string, error) {
	// Get absolute path for better error handling
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get file info for modification time
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to stat config file: %w", err)
	}

	// Check if we have a cached version with the same modification time
	if cached, ok := cachedConfig.Load().(*snapshot); ok && cached != nil {
		if cached.mtime.Equal(fileInfo.ModTime()) {
			return cached.cfg, cached.sha, nil
		}
	}

	// Read file content for SHA computation
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config file: %w", err)
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Load the configuration using the existing pkl-generated code
	cfg, err := config.LoadFromPath(ctx, absPath)
	if err != nil {
		return nil, "", gate.ErrConfigLoad
	}

	// Create and cache the snapshot
	snap := &snapshot{
		cfg:   cfg,
		sha:   hashStr,
		mtime: fileInfo.ModTime(),
	}
	cachedConfig.Store(snap)

	return cfg, hashStr, nil
}
