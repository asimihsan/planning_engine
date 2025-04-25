package config

import (
	"context"
	"fmt"

	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/pkg/config/loader"
)

func Evaluate() (*config.AppConfig, error) {
	cfg, err := config.LoadFromPath(context.Background(), "policy/local/local.pkl")
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}

// EvaluateWithSHA returns the configuration and its SHA
func EvaluateWithSHA() (*config.AppConfig, string, error) {
	return loader.LoadFromPathWithSHA(context.Background(), "policy/local/local.pkl")
}
