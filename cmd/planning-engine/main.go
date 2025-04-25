package main

import (
	"github.com/davecgh/go-spew/spew"

	"github.com/asimihsan/planning_engine/pkg/config"
)

func main() {
	cfg, err := config.Evaluate()
	if err != nil {
		// Handle the error appropriately, e.g., log it and exit
		panic("Failed to load configuration: " + err.Error())
	}

	// Use the configuration as needed
	println("Configuration loaded successfully:", spew.Sdump(cfg))

	// Placeholder for actual implementation
	println("Application started successfully!")
}
