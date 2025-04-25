package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/asimihsan/planning_engine/internal/config"
	"github.com/asimihsan/planning_engine/internal/metrics"
	"github.com/asimihsan/planning_engine/pkg/gate"
)

func main() {
	// Initialize context
	ctx := context.Background()

	// Register Prometheus metrics
	metrics.MustRegister()

	// Load configuration with enhanced loader
	cfg, err := config.LoadFromPath(ctx, "policy/local/local.pkl")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize registry with configuration
	registry := gate.NewFactRegistry()

	// Initialize and register fact providers
	// This would normally include real providers like LevelServer
	// For now, we'll use a placeholder log message

	// Print configuration details
	fmt.Printf("Configuration loaded successfully:\n%s\n", spew.Sdump(cfg))
	fmt.Printf("Fact staleness threshold: %v\n", cfg.FactProviders.MaxStaleness)
	fmt.Printf("Provider timeout: %v\n", cfg.FactProviders.ProviderTimeout)

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		listenAddr := cfg.Prometheus.ListenAddr
		fmt.Printf("Starting metrics server on %s\n", listenAddr)
		if err := http.ListenAndServe(listenAddr, nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Example usage of registry with options
	fmt.Println("Testing registry snapshot with options...")
	opts := gate.SnapshotOpts{
		MaxAge:             cfg.FactProviders.MaxStaleness.GoDuration(),
		PerProviderTimeout: cfg.FactProviders.ProviderTimeout.GoDuration(),
	}

	// In a real application, you would use this with actual providers
	// For now, we'll just demonstrate that the configuration works
	_, err = registry.SnapshotWithOpts(ctx, "test-deployment", "test-stage", opts)
	if err != nil {
		fmt.Printf("Snapshot error (expected with no providers): %v\n", err)
	}

	fmt.Println("Application started successfully!")

	// Keep running to serve metrics
	for {
		time.Sleep(time.Hour)
	}
}
