package levelsrv_mock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
)

// Server provides a mock implementation of the LevelServer API for testing.
type Server struct {
	server       *httptest.Server
	pendingDelta map[string]int // key = deploymentID|stage
	// Note: max_pending_allowed is intentionally removed as it's not provided by the real LevelServer
}

// NewServer creates and starts a new mock LevelServer.
func NewServer() *Server {
	s := &Server{
		pendingDelta: make(map[string]int),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Regex to match paths like /api/deployments/{deploymentID}/stages/{stage}/metrics/{metricName}
		re := regexp.MustCompile(`/api/deployments/([^/]+)/stages/([^/]+)/metrics/([^/]+)`)
		matches := re.FindStringSubmatch(r.URL.Path)

		if matches == nil || len(matches) != 4 {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		deploymentID := matches[1]
		stage := matches[2]
		metricName := matches[3]
		key := fmt.Sprintf("%s|%s", deploymentID, stage)

		switch metricName {
		case "pending_delta":
			if err := json.NewEncoder(w).Encode(map[string]int{"value": s.pendingDelta[key]}); err != nil {
				http.Error(w, "JSON encoding error", http.StatusInternalServerError)
				return
			}
		default:
			// The real LevelServer doesn't provide max_pending_allowed, so we return 404 for anything else
			http.Error(w, "Metric not found", http.StatusNotFound)
		}
	})

	s.server = httptest.NewServer(handler)
	return s
}

// URL returns the URL of the mock server.
func (s *Server) URL() string {
	return s.server.URL
}

// Close shuts down the mock server.
func (s *Server) Close() {
	s.server.Close()
}

// SetPendingDelta sets the value for the pending_delta metric.
func (s *Server) SetPendingDelta(deploymentID, stage string, value int) {
	key := fmt.Sprintf("%s|%s", deploymentID, stage)
	s.pendingDelta[key] = value
}

// WithDefaultValues sets sensible defaults for testing.
func (s *Server) WithDefaultValues() *Server {
	// Default deployment and stage
	defaultKey := "test-deployment|test-stage"
	s.pendingDelta[defaultKey] = 100
	return s
}
