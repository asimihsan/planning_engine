setup:
    mise trust
    mise install

generate:
    mise x -- pkl-gen-go policy/AppConfig.pkl --base-path github.com/asimihsan/planning_engine
    go vet ./...

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
