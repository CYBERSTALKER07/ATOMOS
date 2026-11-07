# Observability & Distributed Tracing Hardening Plan

## TL;DR
Establish enterprise-grade end-to-end observability, W3C TraceContext distributed tracing, and Prometheus monitoring across the PegasusX FMCG platform. This plan upgrades the Go backend (`apps/backend-go`), Python optimizer worker (`services/optimizer-core`), and background daemons to propagate standard W3C `traceparent` headers across HTTP and Kafka transport layers. Additionally, it instruments `services/optimizer-core` with native Prometheus `/metrics`, expands Google Kubernetes Engine (GKE) `PodMonitoring` resources, adjusts zero-trust network policies for metrics scrapers, and defines core Prometheus alerting rules for SLI/SLO compliance.

## Objective
Enable seamless trace correlation and full infrastructure monitoring across all microservices (`backend-go`, `ai-worker`, `optimizer-core`) so that any user or system request carries a correlated W3C trace context from Ingress down to database/solver execution, while all microservices expose verified Prometheus `/metrics` under GKE PodMonitoring and zero-trust NetworkPolicies.

## Non-goals
- Replacing Google Cloud Monitoring or Terraform-managed alert policies with third-party SaaS agents (e.g. Datadog).
- Altering optimizer VRP algorithms or order lifecycle state machine transitions.
- Breaking local docker-compose developer environments or existing `X-Trace-Id` headers (W3C `traceparent` and legacy `X-Trace-Id` must be supported simultaneously in dual-header mode).

## Discovery
- `apps/backend-go/bootstrap/trace_middleware.go`: Custom middleware generates or forwards `X-Trace-Id`, but does not implement standard W3C `traceparent` (`00-{trace_id}-{span_id}-{flags}`) or `propagation.TraceContext`.
- `apps/backend-go/dispatch/optimizerclient/client.go`: Issues `POST /v1/optimizer/solve` without injecting `X-Trace-Id` or W3C `traceparent` into HTTP request headers.
- `services/optimizer-core/server/http_main.py`: Handles `/healthz` and `/ready`, but has no `/metrics` Prometheus endpoint, and does not extract/echo HTTP trace headers.
- `infra/k8s/monitoring/podmonitoring.yaml`: Scrapes `backend-go`, `backend-go-worker`, and `ai-worker`, but lacks a scrape target for `optimizer-core`.
- `infra/k8s/network-policies/optimizer-core-policy.yaml`: Restricts ingress strictly to `backend-go` and `ai-worker`, blocking Prometheus scrapers (`monitoring` / `gke-gmp-system`) on port 8082.
- `apps/backend-go/go.mod`: Already has `go.opentelemetry.io/otel` and `go.opentelemetry.io/otel/trace` available.

## Decisions
- **Decision 1: Dual-Header W3C TraceContext & Legacy Compatibility**: Support standard W3C `traceparent` headers (`00-<32hex>-<16hex>-<2hex>`) as the primary propagation format, while extracting and reflecting `X-Trace-Id` and `X-Request-Id` for backward compatibility with mobile apps and web admin clients.
- **Decision 2: Native Prometheus /metrics Across All Microservices**: Equip `services/optimizer-core` with a lightweight, zero-external-dependency Prometheus text endpoint on `/metrics` exposing process status, request counters, and solver latency distributions.
- **Decision 3: Complete GKE PodMonitoring Coverage**: Add `PodMonitoring` for `optimizer-core` on port 8082, and open ingress for Prometheus scrapers in `optimizer-core-policy.yaml`.
- **Decision 4: Standard PrometheusRule Manifests**: Define standard Prometheus alert rules for high error rates, solver timeouts, consumer lag, and outbox stuck thresholds under `infra/k8s/monitoring/`.

---

## TODOs

### Wave 1: Core OpenTelemetry & W3C TraceContext Propagation

- [x] **Task 1: Standard W3C TraceContext Middleware & Context Bridge**
  - **What to do**: Upgrade `apps/backend-go/bootstrap/trace_middleware.go` to support W3C `traceparent`:
    - Parse incoming `traceparent` header (`00-{trace_id}-{span_id}-{flags}`).
    - Fall back to incoming `X-Trace-Id` or `X-Request-Id`, or generate a cryptographically random 16-byte hex trace ID and 8-byte span ID.
    - Set both `traceparent` and `X-Trace-Id` on response headers.
    - Inject trace ID into request context via `outbox.WithTraceID` and OpenTelemetry context.
    - Add comprehensive unit tests in `apps/backend-go/bootstrap/trace_middleware_test.go`.
  - **What not to do**: Do not discard `X-Trace-Id` or break existing `outbox.WithTraceID` callers.
  - **Files or directories**:
    - `apps/backend-go/bootstrap/trace_middleware.go`
    - `apps/backend-go/bootstrap/trace_middleware_test.go`
  - **References**: W3C TraceContext Specification, `apps/backend-go/outbox/outbox.go`
  - **RED**: `trace_middleware_test.go` fails to extract or format valid W3C `traceparent`.
  - **GREEN**: All tests in `trace_middleware_test.go` pass with 100% W3C format compliance.
  - **Real-surface QA**:
    ```text
    Scenario: Verify W3C traceparent extraction and response headers
    Channel: tmux
    Steps:
      1. cd apps/backend-go && go test -v -run TestTraceMiddleware ./bootstrap
    Expected: "PASS", tests verify extraction of traceparent and generation of valid headers.
    Evidence: Test output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Tests pass cleanly.
  - **Acceptance criteria**: Valid W3C `traceparent` extracted, injected, and correlated with `X-Trace-Id`.
  - **Commit**: YES
    - Message: `feat(telemetry): implement W3C TraceContext propagation in backend trace middleware`
    - Files: `apps/backend-go/bootstrap/trace_middleware.go`, `apps/backend-go/bootstrap/trace_middleware_test.go`
    - Reason: Distributed trace correlation.

- [x] **Task 2: Outbound Service-to-Service Trace Propagation in Optimizer Client**
  - **What to do**: Update `apps/backend-go/dispatch/optimizerclient/client.go` to inject trace context into the outgoing HTTP request:
    - Inject `X-Trace-Id: in.TraceID`.
    - If `in.TraceID` is empty, extract from `outbox.TraceIDFromContext(ctx)`.
    - Construct and inject standard W3C `traceparent` header into `httpReq.Header`.
    - Add unit test in `apps/backend-go/dispatch/optimizerclient/client_test.go` verifying headers are present on outbound requests.
  - **What not to do**: Do not omit `contract.AuthHeader`.
  - **Files or directories**:
    - `apps/backend-go/dispatch/optimizerclient/client.go`
    - `apps/backend-go/dispatch/optimizerclient/client_test.go`
  - **References**: `apps/backend-go/dispatch/optimizerclient/client.go`
  - **RED**: Client request headers lack `X-Trace-Id` and `traceparent`.
  - **GREEN**: Outbound HTTP requests carry `traceparent` and `X-Trace-Id`.
  - **Real-surface QA**:
    ```text
    Scenario: Verify outbound trace header injection in optimizer client
    Channel: tmux
    Steps:
      1. cd apps/backend-go && go test -v ./dispatch/optimizerclient/...
    Expected: "PASS", outbound request inspects and verifies traceparent header.
    Evidence: Test output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Tests pass cleanly.
  - **Acceptance criteria**: Outbound optimizer calls propagate W3C trace context.
  - **Commit**: YES
    - Message: `feat(dispatch): inject W3C traceparent and X-Trace-Id in optimizer client`
    - Files: `apps/backend-go/dispatch/optimizerclient/client.go`, `apps/backend-go/dispatch/optimizerclient/client_test.go`
    - Reason: Service-to-service distributed trace continuity.

---

### Wave 2: Microservice & Worker Observability

- [x] **Task 3: Python Optimizer Core Trace Extraction & Prometheus Metrics**
  - **What to do**: Update `services/optimizer-core/server/http_main.py`:
    - Add `/metrics` handler exporting Prometheus text format:
      - `pegasusx_optimizer_up 1`
      - `pegasusx_optimizer_requests_total{status="200"}`
      - `pegasusx_optimizer_solve_duration_seconds`
    - Extract incoming `traceparent` and `X-Trace-Id` headers in `do_POST`.
    - Return `X-Trace-Id` and `traceparent` on HTTP response headers.
    - Write unit tests in `services/optimizer-core/server/test_observability.py` testing `/metrics` output and trace header reflection.
  - **What not to do**: Do not introduce heavy C-extensions or external network dependencies to Python solver runtime.
  - **Files or directories**:
    - `services/optimizer-core/server/http_main.py`
    - `services/optimizer-core/server/test_observability.py`
  - **References**: `apps/ai-worker/main.go:167-184` (Prometheus text exporter pattern)
  - **RED**: `GET /metrics` returns 404; `POST /v1/optimizer/solve` ignores incoming trace headers.
  - **GREEN**: `GET /metrics` returns 200 with Prometheus gauges; trace headers echoed in response.
  - **Real-surface QA**:
    ```text
    Scenario: Verify optimizer /metrics and trace reflection
    Channel: tmux
    Steps:
      1. python3 services/optimizer-core/server/test_observability.py
    Expected: "OK", metrics return valid text format, trace headers match request.
    Evidence: Script test output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Tests pass cleanly.
  - **Acceptance criteria**: Optimizer exposes Prometheus metrics and correlates W3C traces.
  - **Commit**: YES
    - Message: `feat(optimizer): add Prometheus /metrics endpoint and W3C trace header extraction`
    - Files: `services/optimizer-core/server/http_main.py`, `services/optimizer-core/server/test_observability.py`
    - Reason: Solver observability and trace correlation.

- [x] **Task 4: Kubernetes PodMonitoring & NetworkPolicy Alignment**
  - **What to do**:
    - Add `PodMonitoring` resource for `optimizer-core` to `infra/k8s/monitoring/podmonitoring.yaml` (port: 8082, path: `/metrics`, interval: 30s).
    - Update `infra/k8s/network-policies/optimizer-core-policy.yaml` to allow ingress from `monitoring` and `gke-gmp-system` namespaces on port 8082.
  - **What not to do**: Do not expose port 8082 to public ingress.
  - **Files or directories**:
    - `infra/k8s/monitoring/podmonitoring.yaml`
    - `infra/k8s/network-policies/optimizer-core-policy.yaml`
  - **References**: `infra/k8s/monitoring/podmonitoring.yaml`
  - **RED**: `optimizer-core` missing from `podmonitoring.yaml`; network policy blocks scraper.
  - **GREEN**: `PodMonitoring` resource present and valid; scraper ingress allowed.
  - **Real-surface QA**:
    ```text
    Scenario: Verify PodMonitoring manifest and NetworkPolicy
    Channel: tmux
    Steps:
      1. grep -A 10 "name: optimizer-core" infra/k8s/monitoring/podmonitoring.yaml
      2. grep -A 10 "name: monitoring" infra/k8s/network-policies/optimizer-core-policy.yaml
    Expected: PodMonitoring configured for optimizer-core, and monitoring namespace permitted.
    Evidence: grep outputs.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Manifests formatted and valid.
  - **Acceptance criteria**: All 4 microservices monitored under GKE PodMonitoring and zero-trust policies.
  - **Commit**: YES
    - Message: `feat(k8s): add optimizer-core PodMonitoring and allow scraper network policy ingress`
    - Files: `infra/k8s/monitoring/podmonitoring.yaml`, `infra/k8s/network-policies/optimizer-core-policy.yaml`
    - Reason: Complete cluster metrics scraping coverage.

---

### Wave 3: SRE Prometheus Alerting Rules & SLO Compliance

- [x] **Task 5: Prometheus Alerting Rules for Platform SLIs/SLOs**
  - **What to do**:
    - Create `infra/k8s/monitoring/prometheus-rules.yaml` containing mission-critical PrometheusRule alerts:
      - `PegasusXBackendHighErrorRate`: HTTP 5xx rate > 1% over 5m.
      - `PegasusXBackendHighP95Latency`: P95 HTTP latency > 1000ms over 5m.
      - `PegasusXOptimizerSolverTimeouts`: Optimizer timeouts > 5 over 5m.
      - `PegasusXOutboxRelayStuck`: Outbox relay loop stuck > 60s.
      - `PegasusXKafkaConsumerHighLag`: Kafka consumer lag > 1000 messages.
    - Wire `prometheus-rules.yaml` into `infra/k8s/base/kustomization.yaml`.
  - **What not to do**: Do not create flapping alerts without duration thresholds.
  - **Files or directories**:
    - `infra/k8s/monitoring/prometheus-rules.yaml`
    - `infra/k8s/base/kustomization.yaml`
  - **References**: SRE golden signals (latency, traffic, errors, saturation).
  - **RED**: No PrometheusRule manifest defined in `infra/k8s/monitoring/`.
  - **GREEN**: Valid PrometheusRule manifest added and rendered in base kustomization.
  - **Real-surface QA**:
    ```text
    Scenario: Validate PrometheusRule manifest and base kustomization render
    Channel: tmux
    Steps:
      1. kubectl kustomize --load-restrictor LoadRestrictionsNone infra/k8s/base | grep -A 5 "kind: PrometheusRule"
    Expected: PrometheusRule rendered with active alert conditions.
    Evidence: Rendered YAML output.
    Cleanup: None.
    ```
  - **Cleanup receipt**: Valid YAML manifest integrated.
  - **Acceptance criteria**: 5 critical alert rules defined for SRE reliability.
  - **Commit**: YES
    - Message: `feat(monitoring): define Prometheus alerting rules for SLI/SLO golden signals`
    - Files: `infra/k8s/monitoring/prometheus-rules.yaml`, `infra/k8s/base/kustomization.yaml`
    - Reason: Proactive production incident detection.


---

## Parallel Execution Waves

```text
Wave 1 (OpenTelemetry & Trace Propagation):
  - Task 1: Standard W3C TraceContext Middleware & Context Bridge (backend-go)
  - Task 2: Outbound Service-to-Service Trace Propagation (optimizerclient)

Wave 2 (Microservice Observability & Network Integration):
  - Task 3: Python Optimizer Core Trace Extraction & Prometheus Metrics (optimizer-core)
  - Task 4: Kubernetes PodMonitoring & NetworkPolicy Alignment (infra/k8s)

Wave 3 (SRE Alerting):
  - Task 5: Prometheus Alerting Rules for Platform SLIs/SLOs (infra/k8s)

Critical Path:
Task 1 -> Task 2 -> Task 3 -> Task 4 -> Task 5 -> Final Verification Wave
```

---

## Dependency Matrix

| Task | Depends on | Blocks | Can parallelize with |
|---|---|---|---|
| **Task 1** (Trace Middleware) | none | Task 2 | Task 3, 4 |
| **Task 2** (Optimizer Client Tracing) | Task 1 | Final Wave | Task 3, 4 |
| **Task 3** (Optimizer Metrics & Tracing) | none | Task 4 | Task 1, 2 |
| **Task 4** (PodMonitoring & NetPol) | Task 3 | Task 5 | Task 1, 2 |
| **Task 5** (Prometheus Alert Rules) | Task 4 | Final Wave | Task 1, 2 |

---

## Final Verification Wave

- [x] **1. K8s Manifest & Overlay Lint**:
  - Run YAML validator across all files in `infra/k8s/`.
  - Verify all overlays (`dev`, `staging`, `prod`, `sandbox`) build cleanly via `kubectl kustomize`.
- [x] **2. Automated Backend Test Pass**:
  - `cd apps/backend-go && go test -race ./bootstrap/... ./dispatch/optimizerclient/...` (0 failures, 0 races).
- [x] **3. Optimizer Observability Test Pass**:
  - Execute `python3 services/optimizer-core/server/test_observability.py` (0 failures).
- [x] **4. End-to-End Trace Round-Trip Check**:
  - Verify that a request entering `TraceMiddleware` propagates its W3C traceparent into `optimizerclient` and is reflected back from `optimizer-core`.
- [x] **5. Clean Git Working Tree**:
  - `git status` verifies clean atomic commits with zero uncommitted artifacts.

---

Next: `start-work observability-and-distributed-tracing`
