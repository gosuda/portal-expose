# Tasks: Portal Expose Kubernetes Controller

**Input**: Design documents from `/specs/001-portal-controller/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Standard kubebuilder project layout:
- `api/v1alpha1/`: CRD type definitions
- `controllers/`: Reconciliation logic
- `internal/`: Non-exported utilities
- `config/`: Kubernetes manifests
- `test/`: Integration and E2E tests

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize Kubernetes controller project with kubebuilder

- [X] T001 Initialize Go module with `go mod init github.com/gosuda/portal-expose`
- [X] T002 Run `kubebuilder init --domain portal.gosuda.org --repo github.com/gosuda/portal-expose` to scaffold project
- [X] T003 [P] Create .gitignore for Go projects (bin/, testbin/, cover.out)
- [X] T004 [P] Create initial Makefile targets (build, test, run, docker-build, deploy)
- [X] T005 [P] Set up GitHub Actions CI workflow in .github/workflows/ci.yml for lint, test, build

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core CRD definitions and shared utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 [P] Create PortalExpose CRD using `kubebuilder create api --group portal --version v1alpha1 --kind PortalExpose`
- [X] T007 [P] Create TunnelClass CRD using `kubebuilder create api --group portal --version v1alpha1 --kind TunnelClass`
- [X] T008 [P] Define PortalExpose spec types in api/v1alpha1/portalexpose_types.go (AppSpec, ServiceRef, RelayTarget)
- [X] T009 [P] Define PortalExpose status types in api/v1alpha1/portalexpose_types.go (Phase, PublicURL, TunnelPodStatus, RelayConnectionStatus)
- [X] T010 [P] Define TunnelClass spec types in api/v1alpha1/tunnelclass_types.go (Replicas, Size, NodeSelector, Tolerations)
- [X] T011 [P] Add CRD validation markers to PortalExpose in api/v1alpha1/portalexpose_types.go (kubebuilder:validation)
- [X] T012 [P] Add CRD validation markers to TunnelClass in api/v1alpha1/tunnelclass_types.go (size enum, replicas minimum)
- [X] T013 Run `make manifests` to generate CRD YAML in config/crd/bases/
- [X] T014 Run `make generate` to generate DeepCopy methods in api/v1alpha1/zz_generated.deepcopy.go
- [X] T015 [P] Implement finalizer utilities in internal/util/finalizer.go (AddFinalizer, RemoveFinalizer, HasFinalizer)
- [X] T016 [P] Implement condition utilities in internal/util/conditions.go (SetCondition, FindCondition, IsConditionTrue)
- [X] T017 [P] Create controller test suite setup in controllers/suite_test.go with envtest configuration

**Checkpoint**: Foundation ready - CRDs defined, utilities available, user story implementation can begin

---

## Phase 3: User Story 1 - Create and Expose Service via PortalExpose (Priority: P1) 🎯 MVP

**Goal**: Enable operators to expose Kubernetes Services through Portal relays by creating PortalExpose resources

**Independent Test**: Create PortalExpose → verify tunnel Deployment created → verify status updated with publicURL and phase

### Implementation for User Story 1

- [X] T018 [P] [US1] Implement Deployment builder in internal/tunnel/deployment.go (BuildDeployment function)
- [X] T019 [P] [US1] Implement size-to-resources mapping in internal/tunnel/deployment.go (GetResourcesForSize: small/medium/large)
- [X] T020 [P] [US1] Implement public URL construction in internal/tunnel/status.go (ConstructPublicURL from relay domain + app name)
- [X] T021 [P] [US1] Implement phase computation logic in internal/tunnel/status.go (ComputePhase based on pod readiness)
- [X] T022 [US1] Implement PortalExpose controller reconciliation scaffold in controllers/portalexpose_controller.go
- [X] T023 [US1] Add finalizer management to PortalExpose reconciliation in controllers/portalexpose_controller.go (add on create, remove on delete)
- [X] T024 [US1] Implement Service validation in controllers/portalexpose_controller.go (check Service exists)
- [X] T025 [US1] Implement tunnel Deployment creation in controllers/portalexpose_controller.go (create if not exists)
- [X] T026 [US1] Implement tunnel Deployment ownership in controllers/portalexpose_controller.go (set OwnerReferences)
- [X] T027 [US1] Implement status update logic in controllers/portalexpose_controller.go (phase, publicURL, tunnelPods, conditions)
- [X] T028 [US1] Implement deletion cleanup in controllers/portalexpose_controller.go (delete Deployment, remove finalizer)
- [X] T029 [US1] Add Deployment watch to controller in controllers/portalexpose_controller.go (enqueue PortalExpose on Deployment changes)
- [X] T030 [US1] Add Service watch to controller in controllers/portalexpose_controller.go (enqueue PortalExposes referencing Service)
- [X] T031 [US1] Implement Kubernetes Event emission in controllers/portalexpose_controller.go (Created, Ready, Failed events)
- [X] T032 [US1] Add structured logging to reconciliation in controllers/portalexpose_controller.go (Info, Error logs with context)

### Integration Tests for User Story 1

- [ ] T033 [P] [US1] Write integration test for PortalExpose creation in test/integration/portalexpose_test.go (verify Deployment created)
- [ ] T034 [P] [US1] Write integration test for status updates in test/integration/portalexpose_test.go (verify phase, publicURL populated)
- [ ] T035 [P] [US1] Write integration test for PortalExpose deletion in test/integration/portalexpose_test.go (verify Deployment cleaned up)
- [ ] T036 [P] [US1] Write integration test for Service not found in test/integration/portalexpose_test.go (verify Failed phase)

**Checkpoint**: User Story 1 complete - basic PortalExpose workflow functional and independently testable

---

## Phase 4: User Story 2 - Manage Tunnel Infrastructure via TunnelClass (Priority: P2)

**Goal**: Enable platform teams to define reusable tunnel configurations via TunnelClass resources

**Independent Test**: Create TunnelClass → create PortalExpose referencing it → verify tunnel pods use TunnelClass specs (replicas, resources, nodeSelector)

### Implementation for User Story 2

- [X] T037 [P] [US2] Implement TunnelClass lookup in controllers/portalexpose_controller.go (resolve from spec.tunnelClassName or default)
- [X] T038 [P] [US2] Implement default TunnelClass finder in controllers/portalexpose_controller.go (find by annotation portal.gosuda.org/is-default-class)
- [X] T039 [US2] Update Deployment builder to accept TunnelClass in internal/tunnel/deployment.go (apply replicas, nodeSelector, tolerations)
- [X] T040 [US2] Update reconciliation to pass TunnelClass to Deployment builder in controllers/portalexpose_controller.go
- [X] T041 [US2] Implement TunnelClass controller scaffold in controllers/tunnelclass_controller.go
- [X] T042 [US2] Implement TunnelClass watch to enqueue referencing PortalExposes in controllers/tunnelclass_controller.go
- [X] T043 [US2] Add index for PortalExposes by TunnelClass in controllers/portalexpose_controller.go (SetupWithManager)
- [X] T044 [US2] Update Deployment update logic to trigger rolling updates in controllers/portalexpose_controller.go (detect spec changes)
- [X] T045 [US2] Update status to show Progressing condition during rolling updates in controllers/portalexpose_controller.go

### Integration Tests for User Story 2

- [ ] T046 [P] [US2] Write integration test for TunnelClass reference in test/integration/portalexpose_test.go (verify correct size applied)
- [ ] T047 [P] [US2] Write integration test for default TunnelClass in test/integration/portalexpose_test.go (verify fallback when not specified)
- [ ] T048 [P] [US2] Write integration test for TunnelClass updates in test/integration/tunnelclass_test.go (verify rolling update triggered)
- [ ] T049 [P] [US2] Write integration test for nodeSelector/tolerations in test/integration/portalexpose_test.go (verify pod scheduling constraints)

**Checkpoint**: User Story 2 complete - TunnelClass management functional, PortalExposes inherit infrastructure config

---

## Phase 5: User Story 3 - Handle Multi-Relay Redundancy (Priority: P3)

**Goal**: Enable HA deployments by supporting multiple relay targets in PortalExpose spec

**Independent Test**: Create PortalExpose with multiple relays → verify status shows all relay connection states → simulate relay failure → verify Degraded phase

### Implementation for User Story 3

- [X] T050 [P] [US3] Implement relay status computation in internal/tunnel/status.go (ComputeRelayStatuses from pod annotations/logs)
- [X] T051 [P] [US3] Update Deployment builder to configure multiple relay URLs in internal/tunnel/deployment.go (pass all targets to tunnel args)
- [X] T052 [US3] Update phase computation to handle partial relay failures in internal/tunnel/status.go (Degraded if some relays connected)
- [X] T053 [US3] Update reconciliation to populate status.relay.connected in controllers/portalexpose_controller.go
- [X] T054 [US3] Add RelayConnected condition to status in controllers/portalexpose_controller.go (True if all connected, False with details)
- [X] T055 [US3] Implement relay connection validation in internal/tunnel/validation.go (WSS URL format check)
- [X] T056 [US3] Update status to show per-relay connection states in controllers/portalexpose_controller.go (name, status, connectedAt, lastError)

### Integration Tests for User Story 3

- [ ] T057 [P] [US3] Write integration test for multi-relay setup in test/integration/portalexpose_test.go (verify all relays in status)
- [ ] T058 [P] [US3] Write integration test for partial relay failure in test/integration/portalexpose_test.go (verify Degraded phase)
- [ ] T059 [P] [US3] Write integration test for all relays failed in test/integration/portalexpose_test.go (verify Failed phase)

**Checkpoint**: User Story 3 complete - multi-relay HA functional, partial failures handled gracefully

---

## Phase 6: User Story 4 - Monitor Exposure Status and Health (Priority: P4)

**Goal**: Provide comprehensive status visibility for operators to monitor and debug exposures

**Independent Test**: Create PortalExpose → query status → verify all fields accurate (tunnelPods, relay states, phase, conditions with timestamps)

### Implementation for User Story 4

- [X] T060 [P] [US4] Implement pod readiness computation in internal/tunnel/status.go (count ready vs total pods from Deployment)
- [X] T061 [P] [US4] Implement condition timestamp management in internal/util/conditions.go (update lastTransitionTime only on state change)
- [X] T062 [US4] Update status to populate tunnelPods.ready and tunnelPods.total in controllers/portalexpose_controller.go
- [X] T063 [US4] Add TunnelDeploymentReady condition in controllers/portalexpose_controller.go (True if all pods ready)
- [X] T064 [US4] Add ServiceExists condition in controllers/portalexpose_controller.go (True if Service found, False with error)
- [X] T065 [US4] Add Available condition in controllers/portalexpose_controller.go (True if Ready/Degraded, False if Pending/Failed)
- [X] T066 [US4] Implement Degraded phase for partial pod failures in internal/tunnel/status.go (some pods ready but not all)
- [X] T067 [US4] Add actionable error messages to conditions in controllers/portalexpose_controller.go (include Service name, namespace, etc.)

### Integration Tests for User Story 4

- [ ] T068 [P] [US4] Write integration test for status.tunnelPods accuracy in test/integration/portalexpose_test.go (verify counts match Deployment)
- [ ] T069 [P] [US4] Write integration test for conditions with timestamps in test/integration/portalexpose_test.go (verify lastTransitionTime set)
- [ ] T070 [P] [US4] Write integration test for Degraded phase on pod failure in test/integration/portalexpose_test.go (delete pod, verify status)
- [ ] T071 [P] [US4] Write integration test for actionable error messages in test/integration/portalexpose_test.go (verify condition messages helpful)

**Checkpoint**: All user stories complete - full feature set implemented and independently testable

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T072 [P] Add RBAC role definitions in config/rbac/role.yaml (PortalExposes, TunnelClasses, Deployments, Services, Events)
- [X] T073 [P] Add RBAC role binding in config/rbac/role_binding.yaml
- [X] T074 [P] Create sample PortalExpose manifests in config/samples/ (basic, multi-relay, production)
- [X] T075 [P] Create sample TunnelClass manifests in config/samples/ (default, development, production)
- [X] T076 [P] Add controller manager configuration in config/manager/manager.yaml (replicas, resources, health probes)
- [ ] T077 [P] Implement exponential backoff for transient errors in controllers/portalexpose_controller.go (return error to trigger requeue)
- [ ] T078 [P] Add metrics endpoint configuration in main.go (Prometheus metrics via controller-runtime)
- [ ] T079 [P] Update main.go with structured logging setup (use controller-runtime logger)
- [ ] T080 [P] Add unit tests for Deployment builder in internal/tunnel/deployment_test.go
- [ ] T081 [P] Add unit tests for status computation in internal/tunnel/status_test.go
- [ ] T082 [P] Add unit tests for validation logic in internal/tunnel/validation_test.go
- [X] T083 [P] Add unit tests for finalizer utilities in internal/util/finalizer_test.go
- [ ] T084 [P] Add unit tests for condition utilities in internal/util/conditions_test.go
- [ ] T085 Write E2E test suite setup in test/e2e/suite_test.go (real cluster deployment)
- [ ] T086 Write E2E test for full PortalExpose lifecycle in test/e2e/suite_test.go (create, ready, delete)
- [ ] T087 Add API documentation comments to CRD types in api/v1alpha1/ (kubebuilder markers for doc generation)
- [X] T088 Update README.md with installation instructions, usage examples, architecture overview
- [ ] T089 Run `make test` to verify all unit and integration tests pass
- [ ] T090 Run `make manifests` and commit generated CRD YAMLs
- [ ] T091 Run `gofmt` and `golint` to ensure Go code quality standards

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3 → P4)
- **Polish (Phase 7)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Extends US1 but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Extends US1 but independently testable
- **User Story 4 (P4)**: Can start after Foundational (Phase 2) - Enhances status from US1 but independently testable

### Within Each User Story

- Implementation tasks before integration tests (tests validate implementation)
- Models/types before controllers using them
- Utilities before reconciliation logic using them
- Core reconciliation before watches/indexes
- Status computation before reconciliation status updates

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel (T003, T004, T005)
- All Foundational CRD definition tasks (T006-T012) can run in parallel
- All Foundational utility tasks (T015, T016) can run in parallel after T013-T014
- Within User Story 1: T018-T021 (utilities) can run in parallel
- Within User Story 1: T033-T036 (integration tests) can run in parallel after implementation
- Within User Story 2: T037-T038 can run in parallel
- Within User Story 2: T046-T049 (integration tests) can run in parallel after implementation
- Within User Story 3: T050-T051 can run in parallel
- Within User Story 3: T057-T059 (integration tests) can run in parallel after implementation
- Within User Story 4: T060-T061 can run in parallel
- Within User Story 4: T068-T071 (integration tests) can run in parallel after implementation
- All Polish tasks marked [P] (T072-T084, T087-T091) can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch utility implementations in parallel:
Task: "T018 [P] [US1] Implement Deployment builder in internal/tunnel/deployment.go"
Task: "T019 [P] [US1] Implement size-to-resources mapping in internal/tunnel/deployment.go"
Task: "T020 [P] [US1] Implement public URL construction in internal/tunnel/status.go"
Task: "T021 [P] [US1] Implement phase computation logic in internal/tunnel/status.go"

# After utilities complete, implement controller (sequential - single file):
Task: "T022 [US1] Implement PortalExpose controller reconciliation scaffold"
# ... T023-T032 (controller tasks)

# Launch all integration tests in parallel:
Task: "T033 [P] [US1] Write integration test for PortalExpose creation"
Task: "T034 [P] [US1] Write integration test for status updates"
Task: "T035 [P] [US1] Write integration test for PortalExpose deletion"
Task: "T036 [P] [US1] Write integration test for Service not found"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (T018-T036)
4. **STOP and VALIDATE**: Test PortalExpose create/delete/status workflow
5. Deploy to test cluster and verify basic exposure works

**MVP Deliverable**: Operators can expose Services via PortalExpose, controller manages tunnel Deployments

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 (T018-T036) → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 (T037-T049) → Test independently → Deploy/Demo (TunnelClass support)
4. Add User Story 3 (T050-T059) → Test independently → Deploy/Demo (Multi-relay HA)
5. Add User Story 4 (T060-T071) → Test independently → Deploy/Demo (Rich status)
6. Add Polish (T072-T091) → Full production-ready release

Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (T018-T036)
   - Developer B: User Story 2 (T037-T049)
   - Developer C: User Story 3 (T050-T059)
   - Developer D: User Story 4 (T060-T071)
3. Stories complete and integrate independently
4. Team collaborates on Polish (T072-T091)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Stop at any checkpoint to validate story independently
- Commit after each task or logical group
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- All file paths are relative to repository root
