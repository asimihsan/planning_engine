setup:
    mise trust
    mise install

generate:
    mise x -- pkl-gen-go policy/AppConfig.pkl --base-path github.com/asimihsan/planning_engine
    go vet ./...

run:
    mise x -- go run ./cmd/planning-engine/main.go
