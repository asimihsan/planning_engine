**Overall Goal:**

Develop a robust, extensible, and testable Deployment Gate library/component in Go (Project Name: `planning_engine`) that evaluates deployment safety based on external facts and runtime-configurable OPA policies. This includes:
* An interface-driven, extensible architecture (Open/Closed Principle).
* Centralized fact staleness checks.
* Fail-safe behavior on critical data source errors.
* Atomic hot-reloading of coupled policy and configuration via a manifest.
* Optimized audit logging focusing on relevant data.
* Integration with S3 (for policies/configs) and DynamoDB (for audit logs), validated using LocalStack for testing.
* Basic observability through Prometheus metrics and health endpoints.

**Proposed Code Structure (Illustrative):**

```
planning_engine/ # Renamed project directory
├── cmd/
│   └── gate-cli/         # Example CLI tool for testing/interaction
│       └── main.go
├── configs/              # PKL configuration files & Manifests
│   ├── defaults.pkl
│   ├── localstack.pkl
│   └── manifests/        # Location for policy/config manifests
│       └── latest.json
├── docs/                 # Design docs, usage guides
│   ├── DESIGN.md         # SDD v0.2+ incorporates feedback
│   └── PLAN.md           # This implementation plan
├── examples/             # Example usage as a library
│   └── worker/
│       └── main.go       # Simulates a deployment worker using the gate
├── internal/             # Internal implementations
│   ├── audit/            # AuditLogger implementations
│   │   ├── dynamodb/ logger.go
│   │   └── stdout/ logger.go
│   ├── config/           # Configuration loading logic (PKL + Manifest)
│   │   └── loader.go
│   ├── engine/           # PolicyEngine implementations
│   │   └── opa/ engine.go # Wrapper around OPA Go SDK, potentially exposing decision metadata
│   ├── fact/             # FactProvider implementations
│   │   ├── levelsrv/ provider.go
│   │   └── mock/ provider.go
│   ├── health/           # Health endpoint handler
│   │   └── handler.go
│   ├── metrics/          # Prometheus metrics setup
│   │   └── metrics.go
│   └── policy/           # PolicyProvider implementations
│       ├── file/ provider.go # Reads local manifest/bundle/config
│       └── s3/ provider.go   # Reads manifest from S3, fetches bundle/config
├── pkg/                  # Public library code
│   └── gate/
│       ├── audit.go      # AuditLogger interface
│       ├── config.go     # Structs representing loaded config
│       ├── decision.go   # Decision struct
│       ├── engine.go     # PolicyEngine interface
│       ├── errors.go     # Standardized error types (ErrFactStale, etc.)
│       ├── fact.go       # Fact interface (now includes Timestamp()), Schema struct
│       ├── health.go     # Types related to health status
│       ├── manifest.go   # Struct for the S3 manifest file
│       ├── policy.go     # PolicyBundle, PolicyProvider interfaces
│       └── registry.go   # FactRegistry struct and methods (implements central TTL check)
├── policy/               # OPA policy files and schemas
│   ├── main.rego         # Main policy logic
│   └── input.json        # Schema for expected facts input
├── scripts/              # Helper scripts
│   ├── bundle_policy.sh
│   ├── check_provider_coverage.go # Compile-time static check tool
│   └── setup_localstack.sh # Includes DDB TTL setup
├── tests/                # Integration tests
│   └── integration/
│       ├── gate_test.go
│       └── localstack_setup_test.go # Test setup/teardown for LocalStack
├── go.mod
├── go.sum
└── Makefile              # Build, test, lint, bundle, static-check commands
```

**Milestones / Increments:**

**Milestone 1: Core Interfaces & Basic In-Memory Evaluation**

* **Goal:** Establish core data structures (including timestamped Facts), interfaces, and a minimal OPA evaluation path using local files and mock data. Validate basic policy logic.
* **Key Tasks:**
    * Define interfaces/structs in `pkg/gate/`: `Fact` (with `Timestamp()`), `FactProvider`, `PolicyEngine`, `Decision`, `PolicyBundle`, `PolicyProvider`, `AuditLogger`, core `errors`.
    * Implement basic OPA `PolicyEngine` wrapper (`internal/engine/opa/`).
    * Implement `file://` `PolicyProvider` (initially reads bundle directly, no manifest).
    * Implement `stdout` `AuditLogger`.
    * Implement `mock` `FactProvider` (sets realistic timestamps).
    * Create simple Rego policy (`policy/main.rego`) and `input.json`.
    * Build basic test harness (Go test function).
* **Validation:**
    * Unit tests for interfaces, mocks, providers, logger, engine wrapper.
    * **Add `opa test ./policy/...` step to `Makefile` and CI** to validate Rego logic and track coverage from day one.
    * Integration test (Go test): Load policy from file, evaluate with mock facts, check `Decision` output, verify stdout log.
* **Outcome:** Ability to evaluate a simple policy against hardcoded, timestamped facts loaded from local files, logging to console. Basic OPA testing integrated.

**Milestone 2: Fact Registry, Configuration & Static Checks**

* **Goal:** Introduce dynamic fact collection via the registry, load configuration from PKL, and implement compile-time checks for policy/provider coverage.
* **Key Tasks:**
    * Implement `FactRegistry` (incl. central `FactMaxAge` check using `Fact.Timestamp()`).
    * Implement PKL config loading (`internal/config/loader.go`, `pkg/gate/config.go`).
    * Refactor test harness/example worker to use registry and config.
    * **(Optional Task):** Investigate generating Go structs from `policy/input.json` to aid test writing.
    * **Implement compile-time static check** (`scripts/check_provider_coverage.go`): Parses `policy/input.json`, gets registered provider IDs from Go code, fails build if policy requires facts no provider offers. Integrate into `Makefile` and CI.
* **Validation:**
    * Unit tests for `FactRegistry` (registration, snapshot logic including timestamp/staleness checks).
    * Unit tests for PKL config loading.
    * Integration test: Load config, registry collects from multiple mock providers (testing staleness logic), engine evaluates based on combined facts and config values.
    * CI check: Verify the static provider coverage check passes/fails correctly.
* **Outcome:** Gate dynamically gathers facts, uses external config, enforces staleness centrally. Early detection of missing providers via static analysis in CI.

**Milestone 3: Realistic Fact Provider, Fail-Safe Logic & Metrics**

* **Goal:** Implement `LevelServer` provider with caching/error handling, robust parallelism in registry, and basic provider metrics.
* **Key Tasks:**
    * Implement `LevelServer` `FactProvider` (`internal/fact/levelsrv/`) using HTTP client (target `httptest` mock first), implement caching, return `ErrFactStale`/`ErrFactSourceUnavailable`, set `Timestamp()`.
    * Refine `FactRegistry` to use `errgroup` and configurable `context.WithTimeout` per provider during `Snapshot`.
    * Refine example worker to implement "pause/alert" on specific registry errors.
    * **Add Prometheus histogram metric collection** for `FactProvider` latency (`histogram_vec.WithLabelValues(providerID)`) within provider implementations (`internal/metrics/metrics.go`).
* **Validation:**
    * Unit tests for `LevelServer` provider (caching, errors, timestamps).
    * Integration test (using `httptest`): Verify registry handles healthy, erroring, *and hung* providers correctly (`errgroup` behavior). Verify worker pause/alert logic. Verify fact staleness errors.
    * Verify Prometheus metrics are registered (actual endpoint exposure in M5).
* **Outcome:** Gate interacts robustly with (mocked) external dependency, implements fail-safe pauses, and collects provider latency metrics.

**Milestone 4: LocalStack Integration (Manifest, S3, DDB, Audit Opt.)**

* **Goal:** Integrate with S3/DynamoDB via LocalStack, implement manifest-based policy/config loading, optimize audit logs, and set up DDB TTL.
* **Key Tasks:**
    * Define manifest structure (`pkg/gate/manifest.go`).
    * Implement `s3://` `PolicyProvider` (`internal/policy/s3/`) to read manifest, then fetch corresponding bundle/config from S3 (using AWS SDK Go V2, configurable endpoint).
    * Implement `dynamodb` `AuditLogger` (`internal/audit/dynamodb/`).
        * **Implement "log used facts"**: Modify logger and potentially engine wrapper to log only facts referenced in OPA decision metadata, storing in `usedFactsJSON`.
    * Update PKL config (`configs/localstack.pkl`) for manifest URI, DDB table.
    * Update integration test setup (`tests/integration/localstack_setup_test.go`, `scripts/setup_localstack.sh`) using `testcontainers-go` best practices (retries, caching):
        * Create S3 bucket, DDB table.
        * **Enable TTL on the DDB table** via setup script/testcontainers config.
        * Upload initial manifest, policy bundle, config file.
* **Validation:**
    * Unit tests for S3 manifest provider, DDB logger (mocking AWS SDK).
    * **Integration Test (Key):** Uses LocalStack. Configures S3 Provider, DDB Logger, mock LevelServer Provider. Runs Allow/Deny/Fact Error scenarios. Verifies manifest/policy/config fetched from S3. Verifies *optimized* audit logs (decision/error, `usedFactsJSON`, correct metadata) written to DDB. Verifies DDB items have TTL attribute set (manual check or SDK describe). Simulate S3 updates by uploading a *new manifest file* pointing to new/existing bundle/config versions.
* **Outcome:** Core functionality proven with realistic dependencies (S3, DDB) via LocalStack, including manifest-based loading and optimized auditing.

**Milestone 5: Hot Reload, Metrics Endpoint, Health & Polish**

* **Goal:** Implement atomic hot reloading, expose metrics and health endpoints, refine usability.
* **Key Tasks:**
    * Add polling logic to `S3` `PolicyProvider` to check manifest updates (ETag/version).
    * Implement **atomic swapping** mechanism (e.g., `atomic.Value`) in worker/engine for the *combined* state derived from the manifest (PolicyBundle + Config snapshot).
    * **Expose Prometheus metrics** via HTTP endpoint (`/metrics`) using `internal/metrics` setup (`prometheus/client_golang`).
    * **Implement `/healthz` endpoint** (`internal/health/handler.go`) reporting status and current loaded policy/config versions (SHA/rev).
    * Refine example worker/SDK (`examples/worker/`) for clarity and to demonstrate hot-reloading.
    * Add basic `README.md` documentation.
* **Validation:**
    * Unit tests for polling/refresh logic (if feasible).
    * **Add stress tests with `-race` flag** (`go test -race -run=TestHotReloadStress -count=...`) targeting the hot reload mechanism with concurrent evaluations.
    * Integration test: Start worker process. Perform eval. Update manifest in S3. Wait. Perform another eval. Verify new policy/config used (via decision outcome & audit log). Query `/metrics` endpoint, verify metrics. Query `/healthz` endpoint, verify loaded versions reported correctly.
* **Outcome:** Runtime adaptability and observability proven. PoC library is functional, reasonably robust, and documented for internal use.

**End-to-End Demo Scenario:**

*(Remains largely the same as previous plan, but emphasizes manifest updates and checking `usedFactsJSON` in DDB)*

1.  **Setup:** `make setup-localstack` (starts LocalStack, creates resources, uploads initial manifest v1 pointing to bundle v1/config v1). Start mock LevelServer.
2.  **Run:** `go run examples/worker/main.go --config=configs/localstack.pkl` (worker now loads via manifest specified in PKL). Exposes `/metrics`, `/healthz`.
3.  **Interact & Observe:**
    * **(Allow/Deny/Fail-Safe):** Trigger worker, modify mock LevelServer, stop mock LevelServer. Observe logs, decisions, alerts. Check DDB audit logs verify `outcome`, `policySHA` (v1), `configRev` (v1), and *`usedFactsJSON`*.
    * **(Hot Reload):** Create bundle v2/config v2. Create manifest v2 pointing to these. Run `make upload-manifest MANIFEST=manifest-v2.json`. Wait ~refresh interval. Trigger worker. Observe policy *v2*/config *v2* used in logs/decision. Check DDB audit log for v2 SHAs/Revs. Check `/healthz` output reflects v2 versions.
    * **(Metrics):** `curl localhost:<port>/metrics`. Observe gate metrics.

**End-to-End Test (`tests/integration/gate_test.go`):**

*(Remains conceptually similar but adapted for manifest)*

```go
// Pseudocode for E2E test structure
func TestEndToEndGateScenarios(t *testing.T) {
    // 1. Setup: Start LocalStack (w/ TTL), Mock LevelServer
    // ... (using testcontainers-go)

    // 2. Initial Uploads (Manifest V1 -> Bundle V1, Config V1)
    uploadToS3(t, s3Client, "policy-v1.tar.gz")
    uploadToS3(t, s3Client, "config-v1.pkl")
    uploadManifestToS3(t, s3Client, `{"policy": "policy-v1.tar.gz", "config": "config-v1.pkl", "version": "v1"}`)
    config := loadConfig("localstack.pkl") // Config now just points to manifest URI
    // ... update mock server URL ...

    // 3. Initialize Gate components (S3 Provider reads manifest)

    // 4. Run Scenarios:
    t.Run("Allow Scenario", func(t *testing.T) {
        // ... set mock state, evaluate ...
        verifyAuditLog(t, ddbClient, "allow", "policy-v1-sha", "config-v1-rev", checkUsedFacts)
    })
    // ... Deny Scenario, Fail Safe Scenario ... verifying policy/config v1 revs ...

     t.Run("Hot Reload Scenario", func(t *testing.T) {
        uploadToS3(t, s3Client, "policy-v2.tar.gz") // Upload new bundle
        uploadManifestToS3(t, s3Client, `{"policy": "policy-v2.tar.gz", "config": "config-v1.pkl", "version": "v2"}`) // New manifest points to new policy, old config
        time.Sleep(config.Policy.RefreshInterval + buffer)
        // ... set mock state, evaluate ...
        verifyAuditLog(t, ddbClient, "outcome_for_v2", "policy-v2-sha", "config-v1-rev", checkUsedFacts) // Check policy v2, config v1
    })

    // 5. Verify Metrics / Health endpoints
}
```

**Notes on Other Concerns:**

* **Audit Log Size (S3 Offload):** The primary strategy is logging *used* facts. S3 offload for large audit blobs is a known pattern but **will not** be implemented in this PoC unless item sizes demonstrably exceed DDB limits during testing. It remains a future enhancement option if needed.
* **Configuration/Manifest Management:** The PoC assumes simple file/S3 management. A production system will need a robust process (e.g., GitOps, CI/CD pipelines) for versioning, promoting, and managing PKL config files and S3 manifests across environments. This process is outside the scope of the gate library itself but crucial for operational success.
* **OPA Policy Complexity:** The plan includes `opa test` from M1. As policies grow, maintaining high test coverage and potentially adopting team style guides for Rego will be important.
* **Error Handling Granularity:** The initial "pause/alert" on critical errors (fact source unavailable/stale, policy eval error) is a coarse but safe default. Fine-tuning reactions based on specific error types or sources can be revisited based on operational experience after the PoC.
* **Library vs. Service:** This plan focuses on delivering a Go library (`pkg/gate`). The decision of whether to wrap it in a dedicated gRPC service versus direct integration will be re-evaluated post-PoC based on usage patterns and operational needs. The interface-driven design facilitates either approach.

