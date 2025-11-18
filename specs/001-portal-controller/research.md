# Research: Portal Expose Kubernetes Controller

**Feature**: Portal Expose Kubernetes Controller
**Date**: 2025-01-18
**Purpose**: Technology choices, patterns, and best practices research for controller implementation

## Technology Stack Decisions

### 1. Controller Framework

**Decision**: Kubebuilder v3+ with controller-runtime

**Rationale**:
- Industry standard for building Kubernetes controllers
- Provides scaffolding, code generation, and testing utilities
- Built on top of controller-runtime (shared with Operator SDK)
- Excellent documentation and community support
- Handles CRD generation, RBAC scaffolding, webhook setup automatically
- Used by majority of Kubernetes ecosystem projects

**Alternatives Considered**:
- **Operator SDK**: Very similar to Kubebuilder, slight preference for Helm/Ansible operators (not relevant for Go controllers)
- **Raw client-go**: Too low-level, requires reinventing controller patterns, reconciliation loops, caching
- **Metacontroller**: Declarative approach limits flexibility needed for complex reconciliation logic (rolling updates, status computation)

### 2. CRD API Version

**Decision**: `portal.gosuda.org/v1alpha1`

**Rationale**:
- v1alpha1 signals early API that may change (appropriate for initial release)
- Follows Kubernetes API versioning conventions
- Domain `portal.gosuda.org` aligns with Portal project branding
- Allows future graduation to v1beta1, v1 with backward compatibility support

**Migration Path**:
- v1alpha1 → v1beta1: After API stabilizes in production (6-12 months)
- v1beta1 → v1: After proven stable (12-24 months)
- Conversion webhooks enable serving multiple versions simultaneously

### 3. Status Subresource Strategy

**Decision**: Use status subresource with dedicated status struct

**Rationale**:
- Separates spec (desired state) from status (observed state) - Kubernetes best practice
- Enables optimistic concurrency control (separate resourceVersions for spec vs status)
- Allows users with read-only permissions to view status without spec modification access
- Controller updates status, users update spec - clear separation of responsibility

**Pattern**:
```go
type PortalExposeStatus struct {
    Phase         string
    PublicURL     string
    TunnelPods    TunnelPodStatus
    Relay         RelayStatus
    Conditions    []metav1.Condition
}
```

### 4. Rolling Update Implementation

**Decision**: Leverage Kubernetes Deployment rolling update strategy + reconciliation loop

**Rationale**:
- Controller creates/manages Deployments (not Pods directly)
- Deployment controller handles rolling updates natively when Deployment spec changes
- Controller reconciliation detects TunnelClass/PortalExpose changes and updates Deployment spec
- Kubernetes automatically performs rolling update (RollingUpdate strategy with maxSurge/maxUnavailable)
- No custom rolling update logic needed - rely on battle-tested Kubernetes primitives

**Implementation**:
1. Watch for TunnelClass changes
2. Enqueue all PortalExposes referencing that TunnelClass
3. Reconcile each PortalExpose → update Deployment spec with new TunnelClass values
4. Deployment controller performs rolling update automatically

### 5. Finalizer Strategy

**Decision**: Use finalizer `portal.gosuda.org/cleanup-tunnel-deployment`

**Rationale**:
- Ensures tunnel Deployments are deleted before PortalExpose is removed from etcd
- Prevents orphaned resources if controller crashes during deletion
- Blocks PortalExpose deletion until cleanup completes (critical for proper resource lifecycle)

**Cleanup Flow**:
1. User deletes PortalExpose
2. API server sets deletionTimestamp but doesn't delete (finalizer present)
3. Controller reconciles, sees deletionTimestamp
4. Controller deletes tunnel Deployment
5. Controller removes finalizer from PortalExpose
6. API server completes PortalExpose deletion

**Note**: Owner references also clean up Deployments, but finalizer ensures ordering and allows custom cleanup logic.

### 6. Public URL Construction

**Decision**: Extract relay domain from WSS URL, construct `https://{app.name}.{relay-domain}`

**Rationale**:
- Per clarification Q2: Controller constructs URL, not reported by tunnel pod
- Predictable and immediate (no need to wait for tunnel connection)
- Regex pattern to extract domain from `wss://portal.gosuda.org/relay` → `portal.gosuda.org`
- Public URL becomes `https://{spec.app.name}.portal.gosuda.org`

**Edge Case**: Multiple relays with different domains
- Solution: Use primary relay (first in `spec.relay.targets` list) for public URL
- Document that public URL reflects primary relay; service accessible via all relay domains

### 7. Status Phase State Machine

**Decision**: Four phases - Pending → Ready/Degraded/Failed

**States**:
- **Pending**: Initial state, tunnel Deployment being created, waiting for pods
- **Ready**: All tunnel pods ready, all relays connected
- **Degraded**: Partial failure (some pods not ready OR some relays not connected)
- **Failed**: Complete failure (no pods ready OR no relays connected OR Service not found)

**Transitions**:
```
Pending → Ready:      All pods ready AND all relays connected
Pending → Degraded:   Some pods ready OR some relays connected
Pending → Failed:     No pods ready OR no relays connected OR Service missing

Ready → Degraded:     Pod failure OR relay failure (partial)
Ready → Failed:       All pods fail OR all relays fail

Degraded → Ready:     All pods/relays recovered
Degraded → Failed:    All remaining pods/relays fail

Failed → Pending:     PortalExpose spec updated (trigger reconciliation)
```

### 8. Exponential Backoff for Relay Failures

**Decision**: Use controller-runtime requeue with exponential backoff

**Rationale**:
- controller-runtime provides built-in rate limiting and backoff via `controller.Options.RateLimiter`
- Default backoff: 5ms → 10ms → 20ms → ... up to max delay (typically 1000s / 16.7 minutes)
- Requeue failed reconciliations automatically
- Per-resource backoff (doesn't slow down other PortalExposes)

**Implementation**:
```go
return ctrl.Result{Requeue: true}, err // Triggers exponential backoff
```

For relay connection failures:
- Don't mark as permanent error
- Return transient error to trigger requeue with backoff
- Status shows "Degraded" or "Failed" with condition explaining relay unreachable
- Continues retrying in background

### 9. Condition Types

**Decision**: Use standard Kubernetes condition types + custom conditions

**Standard Conditions**:
- `Available`: Overall PortalExpose health (Ready/Degraded state)
- `Progressing`: Tunnel Deployment rollout in progress

**Custom Conditions**:
- `TunnelDeploymentReady`: Tunnel pods at desired replica count
- `RelayConnected`: All relay targets connected (True) vs partial (False with warning message)
- `ServiceExists`: Referenced Kubernetes Service found

**Rationale**:
- Conditions provide granular state visibility beyond simple phase
- Standard conditions recognized by Kubernetes tooling (kubectl, dashboards)
- Custom conditions convey domain-specific state (relay connectivity)
- Timestamps and messages enable debugging

### 10. RBAC Permissions

**Decision**: Minimal RBAC - only required resources

**Required Permissions**:
- **portalexposes.portal.gosuda.org**: create, get, list, watch, update, patch, delete (full CRUD)
- **portalexposes/status**: get, update, patch (status subresource)
- **tunnelclasses.portal.gosuda.org**: get, list, watch (read-only, controllers don't modify TunnelClasses)
- **deployments.apps**: create, get, list, watch, update, patch, delete (manage tunnel Deployments)
- **services**: get, list, watch (validate Service exists, read-only)
- **events**: create, patch (emit Kubernetes Events)

**Not Needed**:
- cluster-admin
- Secrets (no credentials managed by controller - tunnel pods may need Secrets but controller doesn't)
- Pods (managed via Deployments, not directly)
- Ingress, ConfigMaps, etc. (out of scope)

### 11. Testing Strategy

**Unit Tests**:
- Test reconciliation logic with fake client
- Test status computation functions (phase transitions, condition generation)
- Test validation logic (TunnelClass size values, relay URL format)
- Test utility functions (finalizer management, condition helpers)

**Integration Tests** (envtest):
- Create PortalExpose → verify Deployment created with correct spec
- Update TunnelClass → verify Deployment updated (rolling update triggered)
- Delete PortalExpose → verify Deployment removed
- Partial pod failure → verify phase transitions to Degraded
- Service not found → verify phase transitions to Failed

**E2E Tests** (optional, requires real cluster):
- Deploy controller to cluster
- Create real PortalExpose with Service
- Verify tunnel pods actually connect to Portal relay (requires live relay)
- Verify public URL accessibility

**Testing Framework**:
- Go testing package (standard library)
- envtest (kubebuilder integration testing, runs real API server locally)
- Ginkgo/Gomega optional (BDD-style, nice but not required)

### 12. Observability Implementation

**Structured Logging**:
- Use controller-runtime's built-in logger (based on logr interface)
- Log at appropriate levels: Info for state changes, Debug for reconciliation details, Error for failures
- Include context: PortalExpose name/namespace, phase transitions, errors

**Metrics** (optional for MVP):
- Prometheus metrics via controller-runtime metrics endpoint
- Standard controller metrics (reconciliation duration, queue depth, errors)
- Custom metrics: active PortalExposes by phase, relay connection success rate

**Events**:
- Emit Events for user-visible state changes
- Event types: Normal (Created, Ready, Updated), Warning (Degraded, ServiceNotFound), Error (Failed)

## Best Practices Applied

### Controller Pattern Best Practices

1. **Idempotent Reconciliation**: Same inputs always produce same outputs, safe to retry
2. **Level-Triggered**: React to current state, not edge transitions (watch events may be missed)
3. **Status Reflects Reality**: Status shows what controller observes, not what it hopes for
4. **Owner References**: Set PortalExpose as owner of tunnel Deployment for automatic cleanup
5. **Finalizers**: Ensure cleanup before deletion
6. **Watches**: Watch PortalExposes, TunnelClasses, and Deployments (to detect external changes)

### Kubernetes API Conventions

1. **Spec/Status Separation**: Spec is desired state (user input), Status is observed state (controller output)
2. **Conditions**: Use metav1.Condition for granular state signaling
3. **Resource Naming**: Use GenerateName or construct deterministic names (`{portalexpose-name}-tunnel`)
4. **Labels/Annotations**: Add standard labels (app.kubernetes.io/name, app.kubernetes.io/component)
5. **API Versioning**: Start with v1alpha1, graduate through v1beta1 to v1

### Portal-Specific Patterns

1. **Relay Failover**: Tunnel pods connect to all relays simultaneously (not failover-based)
2. **App Name as Subdomain**: `spec.app.name` becomes subdomain (validate DNS-safe characters)
3. **TunnelClass Inheritance**: PortalExpose references TunnelClass by name (loose coupling)
4. **Default TunnelClass**: Annotation `portal.gosuda.org/is-default-class: "true"` marks default

## Open Questions (Resolved)

None. All technical decisions are informed by specification requirements and clarifications.

## References

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [controller-runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Writing Controllers Guide](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)
- [Portal Project](https://github.com/gosuda/portal) - Relay protocol and architecture
