//go:build tools
// +build tools

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Schema represents a JSON Schema object from input.json
type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

// Property represents a JSON Schema property
type Property struct {
	Type string `json:"type"`
}

// getRequiredFacts parses the policy input schema to get required fact IDs
func getRequiredFacts(schemaPath string) (map[string]bool, error) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	requiredFacts := make(map[string]bool)
	for _, factID := range schema.Required {
		requiredFacts[factID] = false // not found yet
	}

	return requiredFacts, nil
}

// getProvidedFacts scans the codebase for fact provider implementations
func getProvidedFacts() (map[string]bool, error) {
	providedFacts := make(map[string]bool)

	// Walk through the internal/fact directory to find provider implementations
	err := filepath.Walk("internal/fact", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process Go files
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			// Parse the Go file
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return err
			}

			// Look for provider implementations
			ast.Inspect(file, func(n ast.Node) bool {
				// Look for calls to functions like NewProvider with a string parameter
				// or struct literals with ID field
				if call, ok := n.(*ast.CallExpr); ok {
					// Check for function calls like NewProvider("fact_id", ...)
					if fun, ok := call.Fun.(*ast.Ident); ok && strings.Contains(fun.Name, "Provider") {
						if len(call.Args) > 0 {
							if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
								// Extract the fact ID from the string literal
								factID := strings.Trim(lit.Value, "\"")
								providedFacts[factID] = true
							}
						}
					}
				}

				// Look for Describe method implementations
				if funcDecl, ok := n.(*ast.FuncDecl); ok {
					if funcDecl.Name.Name == "Describe" && funcDecl.Recv != nil {
						// This is a Describe method, look at its body for Schema{ID: "..."}
						if funcDecl.Body != nil {
							ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
								if ret, ok := n.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
									// Get the source code for the return statement to extract the ID field
									// This is a simplified approach - ideally we'd properly walk the AST
									src := make([]byte, 0)
									if file, err := os.ReadFile(path); err == nil {
										src = file
									}

									if len(src) > 0 {
										// Extract the Schema instantiation using a regex
										// This is a simplification, but works for common patterns
										r := regexp.MustCompile(`Schema\{ID:\s*"([^"]+)"`)
										if ret.Results[0] != nil {
											pos := fset.Position(ret.Results[0].Pos())
											end := fset.Position(ret.Results[0].End())

											if pos.Offset >= 0 && end.Offset <= len(src) {
												returnText := string(src[pos.Offset:end.Offset])
												matches := r.FindStringSubmatch(returnText)
												if len(matches) > 1 {
													factID := matches[1]
													providedFacts[factID] = true
												}
											}
										}
									}
								}
								return true
							})
						}
					}
				}
				return true
			})
		}
		return nil
	})

	return providedFacts, err
}

func main() {
	// Get required facts from the policy schema
	requiredFacts, err := getRequiredFacts("policy/rego/input.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting required facts: %v\n", err)
		os.Exit(1)
	}

	// Get provided facts from the codebase
	providedFacts, err := getProvidedFacts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning providers: %v\n", err)
		os.Exit(1)
	}

	// Verify that all required facts have providers
	missingFacts := []string{}
	for factID := range requiredFacts {
		if !providedFacts[factID] {
			missingFacts = append(missingFacts, factID)
		}
	}

	// Report missing facts
	if len(missingFacts) > 0 {
		fmt.Fprintf(os.Stderr, "ERROR: The following required facts have no provider implementations:\n")
		for _, factID := range missingFacts {
			fmt.Fprintf(os.Stderr, "  - %s\n", factID)
		}
		os.Exit(1)
	}

	fmt.Println("SUCCESS: All required facts have provider implementations.")
}
