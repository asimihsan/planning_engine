setup:
    mise trust
    mise install

clean:
    rm -f internal/config/*
    touch internal/config/.gitkeep

generate: clean
    mise x -- pkl-gen-go policy/AppConfig.pkl --base-path github.com/asimihsan/planning_engine
    go vet ./...
    # Run the static provider coverage check
    go run ./scripts/check_provider_coverage.go

run:
    mise x -- go run ./cmd/planning-engine/main.go

lint:
    mise x -- gofumpt -d -e .
    mise x -- golangci-lint run ./...

lint-fix:
    mise x -- gofumpt -w .
    mise x -- golangci-lint run --fix ./...

test:
    mise x -- go test -race ./...
    # Run OPA policy tests
    mise x -- opa test ./policy/rego

static-check:
    go run ./scripts/check_provider_coverage.go

check-all: generate lint test static-check
