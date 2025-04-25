package levelsrv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Collect(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/deployments/test-deployment/stages/test-stage/metrics/pending_delta":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{"value": 100}) //nolint:errcheck
		case "/api/deployments/test-deployment/stages/test-stage/metrics/max_pending_allowed":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{"value": 500}) //nolint:errcheck
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name        string
		factID      string
		wantValue   int
		wantErr     bool
		errExpected error
	}{
		{
			name:      "pending_delta fact",
			factID:    "pending_delta",
			wantValue: 100,
			wantErr:   false,
		},
		{
			name:      "max_pending_allowed fact",
			factID:    "max_pending_allowed",
			wantValue: 500,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProvider(tt.factID, server.URL, 500*time.Millisecond, "Test description")

			// Collect fact
			fact, err := p.Collect(context.Background(), "test-deployment", "test-stage")
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errExpected != nil {
					assert.ErrorIs(t, err, tt.errExpected)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.factID, fact.ID())
			assert.Equal(t, tt.wantValue, fact.Value())
			assert.True(t, time.Since(fact.Timestamp()) < time.Second)

			// Test caching
			firstTimestamp := fact.Timestamp()
			time.Sleep(100 * time.Millisecond) // Wait a bit but less than TTL

			// Get from cache
			cachedFact, err := p.Collect(context.Background(), "test-deployment", "test-stage")
			require.NoError(t, err)
			assert.Equal(t, firstTimestamp, cachedFact.Timestamp()) // Should be same timestamp from cache
		})
	}
}

func TestProvider_CacheExpiry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"value": callCount}) //nolint:errcheck
	}))
	defer server.Close()

	// Create provider with very short TTL
	p := NewProvider("test_fact", server.URL, 50*time.Millisecond, "Test description")

	// First call
	fact1, err := p.Collect(context.Background(), "test-deployment", "test-stage")
	require.NoError(t, err)
	assert.Equal(t, 1, fact1.Value())

	// Immediate second call should use cache
	fact2, err := p.Collect(context.Background(), "test-deployment", "test-stage")
	require.NoError(t, err)
	assert.Equal(t, 1, fact2.Value()) // Same value from cache
	assert.Equal(t, fact1.Timestamp(), fact2.Timestamp())

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Third call should get fresh data
	fact3, err := p.Collect(context.Background(), "test-deployment", "test-stage")
	require.NoError(t, err)
	assert.Equal(t, 2, fact3.Value()) // New value after cache expiry
	assert.NotEqual(t, fact1.Timestamp(), fact3.Timestamp())
}

func TestProvider_ErrorHandling(t *testing.T) {
	// Create a server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := NewProvider("test_fact", server.URL, 1*time.Second, "Test description")

	// Server error should return ErrFactSourceUnavailable
	_, err := p.Collect(context.Background(), "test-deployment", "test-stage")
	assert.Error(t, err)
	assert.True(t, gate.IsWrappingError(err, gate.ErrFactSourceUnavailable))

	// Invalid URL should return error
	badProvider := NewProvider("bad_fact", "http://invalid-url-that-wont-resolve", 1*time.Second, "Bad URL")
	_, err = badProvider.Collect(context.Background(), "test-deployment", "test-stage")
	assert.Error(t, err)
	assert.True(t, gate.IsWrappingError(err, gate.ErrFactSourceUnavailable))
}
