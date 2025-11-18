# Implementation Plan: Portal Expose Kubernetes Controller

**Branch**: `001-portal-controller` | **Date**: 2025-01-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-portal-controller/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build a Kubernetes controller that manages PortalExpose and TunnelClass custom resources to automatically expose cluster services to the internet through Portal relays. The controller watches CRDs, deploys tunnel pods with configurations derived from TunnelClass specifications, manages relay connections, and reports status including public URLs, pod health, and relay connectivity. Key capabilities include rolling updates for zero-downtime configuration changes, graceful degradation with "Degraded" phase for partial failures, and automatic resource cleanup via owner references and finalizers.

## Technical Context

**Language/Version**: Go 1.21+ (Kubernetes ecosystem standard)
**Primary Dependencies**: controller-runtime, client-go, kubebuilder (Kubernetes controller framework)
**Storage**: Kubernetes etcd (via API server for CRD storage) - no external database required
**Testing**: Go testing framework, envtest (Kubernetes integration testing), Ginkgo/Gomega (optional BDD-style)
**Target Platform**: Linux (Kubernetes control plane, typically runs as Deployment in cluster)
**Project Type**: Single Kubernetes controller project (standard kubebuilder layout)
**Performance Goals**:
  - Reconciliation latency: <500ms per PortalExpose resource
  - Support 100+ PortalExpose resources without degradation
  - Status updates within 5 seconds of state changes
  - Tunnel pod deployment within 30 seconds
**Constraints**:
  - Must not modify user-specified Service resources
  - Controller image size <200MB for fast pod startup
  - Memory footprint <100MB baseline + <1MB per managed PortalExpose
  - RBAC must follow least-privilege (no cluster-admin)
**Scale/Scope**:
  - Target: 100-500 PortalExpose resources per cluster
  - Each PortalExpose manages 1 Deployment (1-10 tunnel pod replicas typical)
  - Controller codebase estimated ~5k-10k LOC (controllers + CRD types + utilities)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Kubernetes-Native Design ✅ PASS
- ✅ CRDs (PortalExpose, TunnelClass) are primary user interface
- ✅ Controller follows watch-reconcile-update status loop pattern
- ✅ Declarative YAML specifications (no imperative commands)
- ✅ Status fields reflect actual state (phase, pod counts, relay connectivity)
- ✅ All configuration expressible as manifests

### II. Security by Default ✅ PASS
- ✅ TLS/WSS enforced for tunnel communication (FR-022)
- ✅ Controller manages tunnel image/version, not user-specifiable (FR-021)
- ✅ RBAC least-privilege approach (Technical Context constraint)
- ✅ Secrets managed via Kubernetes Secret resources (assumption)
- ✅ No credential exposure in logs/status (constitution requirement)

### III. Clear Separation of Concerns ✅ PASS
- ✅ TunnelClass defines HOW tunnels run (size, replicas, scheduling)
- ✅ PortalExpose defines WHAT to expose (service, relay targets)
- ✅ Controller manages lifecycle and reconciliation
- ✅ Users cannot control security settings or tunnel internals

### IV. Observable and Debuggable ✅ PASS
- ✅ Status phases: Pending/Ready/Degraded/Failed (FR-009)
- ✅ Conditions with timestamps and messages (FR-013)
- ✅ Relay connectivity status per target (FR-012)
- ✅ Pod readiness exposed in status.tunnelPods (FR-011)
- ✅ Public URLs populated in status (FR-010)
- ✅ Kubernetes Events for state transitions (FR-018)
- ✅ Structured logging for reconciliation (FR-019)

### V. Graceful Degradation and Resilience ✅ PASS
- ✅ Tunnel pod failures trigger auto-reconciliation (Degraded phase)
- ✅ Relay connection failures use exponential backoff (FR-020)
- ✅ Partial relay connectivity shows Degraded, not Failed (clarification Q4)
- ✅ Resource deletion cleanup via owner references (FR-014) and finalizers (FR-017)
- ✅ Controller crash recovery without orphaned resources (SC-009)
- ✅ Idempotent reconciliation (controller pattern)

### Quality Standards Gates ✅ PASS
- ✅ Unit tests for reconciliation logic (constitution requirement)
- ✅ Integration tests for CRD workflows (constitution requirement)
- ✅ E2E tests for tunnel connectivity (constitution requirement)
- ✅ Tests runnable via `make test` (constitution requirement)
- ✅ Go conventions (gofmt, golint) (constitution requirement)
- ✅ Controller-runtime patterns (constitution requirement)
- ✅ Actionable error messages (SC-010)

**GATE RESULT**: ✅ ALL CHECKS PASS - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
portal-expose/
├── api/
│   └── v1alpha1/
│       ├── portalexpose_types.go       # PortalExpose CRD definition
│       ├── tunnelclass_types.go        # TunnelClass CRD definition
│       ├── groupversion_info.go        # API group/version metadata
│       └── zz_generated.deepcopy.go    # Generated deep copy methods
├── controllers/
│   ├── portalexpose_controller.go      # PortalExpose reconciliation logic
│   ├── tunnelclass_controller.go       # TunnelClass reconciliation logic
│   └── suite_test.go                   # Controller test suite setup
├── internal/
│   ├── tunnel/
│   │   ├── deployment.go               # Tunnel Deployment generation
│   │   ├── status.go                   # Status computation helpers
│   │   └── validation.go               # PortalExpose/TunnelClass validation
│   └── util/
│       ├── finalizer.go                # Finalizer management utilities
│       └── conditions.go               # Condition type helpers
├── config/
│   ├── crd/
│   │   └── bases/
│   │       ├── portal.gosuda.org_portalexposes.yaml
│   │       └── portal.gosuda.org_tunnelclasses.yaml
│   ├── rbac/
│   │   ├── role.yaml                   # RBAC permissions
│   │   └── role_binding.yaml
│   ├── manager/
│   │   ├── manager.yaml                # Controller Deployment manifest
│   │   └── kustomization.yaml
│   └── samples/                        # Example CRD instances
├── test/
│   ├── e2e/
│   │   └── suite_test.go               # End-to-end test suite
│   └── integration/
│       ├── portalexpose_test.go        # PortalExpose integration tests
│       └── tunnelclass_test.go         # TunnelClass integration tests
├── main.go                             # Controller entry point
├── Makefile                            # Build, test, deploy targets
├── go.mod
└── go.sum
```

**Structure Decision**: Standard kubebuilder project layout used. This is the Kubernetes ecosystem convention for controller projects, providing:
- `api/v1alpha1/`: CRD type definitions (auto-generates YAML manifests)
- `controllers/`: Reconciliation logic for each CRD
- `internal/`: Non-exported utilities and business logic
- `config/`: Kubernetes manifests (CRDs, RBAC, deployment)
- `test/`: Integration and E2E tests (unit tests colocated with source files)
- Root-level `main.go` bootstraps controller manager

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations. All complexity is justified by Kubernetes controller requirements and constitution compliance.
