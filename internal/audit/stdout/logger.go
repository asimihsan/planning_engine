package stdout

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/asimihsan/planning_engine/pkg/gate"
)

// Logger implements gate.AuditLogger with output to stdout.
type Logger struct{}

// New creates a new stdout logger.
func New() *Logger {
	return &Logger{}
}

// LogDecision implements gate.AuditLogger.
func (l *Logger) LogDecision(ctx context.Context, input map[string]any, decision gate.Decision, policyID, configID string, evalDuration time.Duration) error {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		inputJSON = []byte(fmt.Sprintf("error marshaling input: %v", err))
	}

	log.Printf("[AUDIT DECISION] PolicyID: %s, ConfigID: %s, Allow: %v, Reasons: %v, Duration: %s, Input: %s\n",
		policyID, configID, decision.Allow, decision.DenyReasons, evalDuration, string(inputJSON))

	return nil
}

// LogSystemError implements gate.AuditLogger.
func (l *Logger) LogSystemError(ctx context.Context, systemError error, deploymentID, stage, policyID, configID string) error {
	log.Printf("[AUDIT SYSTEM ERROR] DeploymentID: %s, Stage: %s, PolicyID: %s, ConfigID: %s, Error: %v\n",
		deploymentID, stage, policyID, configID, systemError)

	return nil
}
