# Feature Specification: Portal Expose Kubernetes Controller

**Feature Branch**: `001-portal-controller`
**Created**: 2025-01-18
**Status**: Draft
**Input**: User description: "i need PortalExpose's controller. this is kubernetes controller that manage PortalExpose, TunnelClass. plz ref README.md and this repo "https://github.com/gosuda/portal""

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Expose Service via PortalExpose (Priority: P1)

A platform operator wants to expose an internal Kubernetes service to the public internet through Portal relays. They create a PortalExpose custom resource pointing to their service, and the controller automatically deploys tunnel pods that establish connections to Portal relays, making the service accessible via a public URL.

**Why this priority**: This is the core value proposition - enabling users to expose services without manual tunnel management. Without this, the controller provides no value.

**Independent Test**: Can be fully tested by creating a PortalExpose resource referencing an existing Kubernetes Service, then verifying that tunnel pods are deployed and the service becomes accessible via the public URL reported in the PortalExpose status.

**Acceptance Scenarios**:

1. **Given** a Kubernetes Service named "my-app" exists on port 8080, **When** I create a PortalExpose resource referencing "my-app" with relay URL "wss://portal.gosuda.org/relay", **Then** the controller deploys tunnel pods and updates the PortalExpose status with phase "Ready" and a public URL
2. **Given** a PortalExpose resource is in "Ready" state, **When** I access the public URL from outside the cluster, **Then** my request reaches the internal Kubernetes Service
3. **Given** a PortalExpose resource exists, **When** I delete it, **Then** the controller removes all associated tunnel pods and cleans up resources

---

### User Story 2 - Manage Tunnel Infrastructure via TunnelClass (Priority: P2)

A platform administrator wants to define reusable tunnel configurations for different workload tiers (development, production, high-traffic). They create TunnelClass resources specifying resource sizes, replica counts, and scheduling constraints. PortalExpose resources reference these TunnelClasses to inherit the appropriate tunnel configuration.

**Why this priority**: Enables separation of infrastructure concerns (how tunnels run) from application concerns (what to expose), allowing platform teams and app teams to work independently. Required for production-grade deployments but not for basic functionality.

**Independent Test**: Can be fully tested by creating a TunnelClass with specific size/replicas/node selectors, then creating a PortalExpose that references it, and verifying the deployed tunnel pods match the TunnelClass specifications (resource requests/limits, replica count, node placement).

**Acceptance Scenarios**:

1. **Given** a TunnelClass named "production" exists with size=large and replicas=3, **When** I create a PortalExpose referencing tunnelClassName "production", **Then** the controller deploys 3 tunnel pods with large resource allocations
2. **Given** a TunnelClass with nodeSelector specifying "workload-type: tunnel", **When** a PortalExpose references this TunnelClass, **Then** tunnel pods are scheduled only on nodes with that label
3. **Given** no TunnelClass is specified in a PortalExpose, **When** the controller reconciles it, **Then** it uses the TunnelClass marked with annotation "portal.gosuda.org/is-default-class: true"

---

### User Story 3 - Handle Multi-Relay Redundancy (Priority: P3)

An operator wants high availability for their exposed service by connecting to multiple Portal relay endpoints. They specify multiple relay targets in the PortalExpose spec, and the controller ensures tunnel pods connect to all of them, providing redundancy if one relay becomes unavailable.

**Why this priority**: Important for production HA scenarios and demonstrated in the README examples with multiple relays. This enables the resilience needed for production deployments and is a core feature of the system architecture.

**Independent Test**: Can be fully tested by creating a PortalExpose with multiple relay targets, then verifying that status.relay.connected shows all relays as connected, and that the service remains accessible even if one relay connection fails.

**Acceptance Scenarios**:

1. **Given** a PortalExpose specifies relay targets for both "wss://portal.gosuda.org/relay" and "wss://portal.thumbgo.kr/relay", **When** the controller reconciles it, **Then** tunnel pods establish connections to both relays
2. **Given** a PortalExpose with 3 relay targets where 1 fails to connect, **When** checking the status, **Then** status shows 2/3 relays connected and phase shows "Degraded" with a condition warning about partial connectivity
3. **Given** all relay targets fail to connect, **When** the controller updates status, **Then** phase shows "Failed" and conditions explain no relay connections could be established

---

### User Story 4 - Monitor Exposure Status and Health (Priority: P4)

An operator wants to monitor the health of their exposed services. They check the PortalExpose status fields to see tunnel pod readiness, relay connection states, and any error conditions. The status provides visibility into whether the exposure is working correctly.

**Why this priority**: Enhances operability and debugging but the exposure can function without rich status reporting. Users can verify functionality by testing the public URL directly. While helpful for production operations, basic status is provided by P3 (relay connectivity tracking).

**Independent Test**: Can be fully tested by creating a PortalExpose and querying its status to verify it shows accurate information about tunnel pod counts (ready/total), relay connection states (connected/disconnected), phase (Pending/Ready/Failed), and conditions with timestamps.

**Acceptance Scenarios**:

1. **Given** a PortalExpose with 2 tunnel replicas is reconciled, **When** both pods are ready, **Then** status shows tunnelPods.ready=2 and tunnelPods.total=2
2. **Given** a PortalExpose specifies 2 relay targets, **When** connections are established, **Then** status.relay.connected shows both relays with status "Connected" and timestamps
3. **Given** a PortalExpose references a non-existent Service, **When** the controller reconciles it, **Then** status.phase shows "Failed" and conditions include an error message explaining the Service was not found
4. **Given** a PortalExpose with 3 tunnel replicas where 1 pod crashes, **When** the controller updates status, **Then** status.phase shows "Degraded" and status.tunnelPods shows ready=2/total=3

---

### Edge Cases

- What happens when the referenced Kubernetes Service doesn't exist at PortalExpose creation time?
- What happens when the Service is deleted while a PortalExpose referencing it exists?
- What happens when a TunnelClass is modified after PortalExposes are already using it? (Controller performs rolling update of tunnel Deployments to apply new configuration without disruption)
- What happens when a TunnelClass is deleted while PortalExposes reference it?
- What happens when relay URLs are unreachable or invalid (malformed WSS URLs)?
- What happens when tunnel pods crash or are evicted due to node pressure? (Phase transitions to "Degraded" if some but not all pods fail; controller reconciles to restore desired replica count)
- What happens when multiple PortalExposes try to use the same app name (subdomain conflicts)?
- What happens when network policies block tunnel pod communication?
- What happens when no default TunnelClass exists and PortalExpose doesn't specify one?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Controller MUST watch for PortalExpose custom resources and reconcile them to desired state
- **FR-002**: Controller MUST watch for TunnelClass custom resources and apply their configurations to tunnel deployments using rolling updates when TunnelClass is modified
- **FR-003**: Controller MUST create Kubernetes Deployments for tunnel pods when a PortalExpose is created
- **FR-004**: Controller MUST configure tunnel pods with the Service name, port, and relay URLs from the PortalExpose spec
- **FR-005**: Controller MUST apply resource requests/limits to tunnel pods based on the TunnelClass size field (small/medium/large)
- **FR-006**: Controller MUST set tunnel pod replica count based on TunnelClass replicas field
- **FR-007**: Controller MUST apply nodeSelector and tolerations from TunnelClass to tunnel pod specs
- **FR-008**: Controller MUST use a default TunnelClass when PortalExpose doesn't specify tunnelClassName
- **FR-009**: Controller MUST update PortalExpose status.phase to reflect current state (Pending/Ready/Degraded/Failed) where Degraded indicates partial pod or relay failures
- **FR-010**: Controller MUST populate PortalExpose status.publicURL with the accessible endpoint constructed from app name and relay domain (e.g., `https://{app.name}.{relay-domain}`)
- **FR-011**: Controller MUST update PortalExpose status.tunnelPods with ready and total counts
- **FR-012**: Controller MUST update PortalExpose status.relay.connected with per-relay connection states
- **FR-013**: Controller MUST set status.conditions with typed conditions (TunnelDeployed, RelayConnected, etc.)
- **FR-014**: Controller MUST delete tunnel Deployments when a PortalExpose is deleted (garbage collection via owner references)
- **FR-015**: Controller MUST validate that referenced Services exist before marking PortalExpose as Ready
- **FR-016**: Controller MUST handle multiple relay targets by configuring tunnel pods to connect to all of them
- **FR-017**: Controller MUST use finalizers to ensure clean deletion of resources
- **FR-018**: Controller MUST generate Kubernetes Events for significant state transitions (created, failed, ready, deleted)
- **FR-019**: Controller MUST use structured logging for all reconciliation operations
- **FR-020**: Controller MUST implement exponential backoff for retry logic on transient failures
- **FR-021**: Controller MUST enforce that tunnel container image and version are controlled by the controller, not user-specifiable
- **FR-022**: Controller MUST ensure all tunnel communication uses TLS encryption (WSS protocol)
- **FR-023**: Controller MUST follow Kubernetes controller patterns (watch-reconcile-update status loop)
- **FR-024**: Controller MUST apply PortalExpose spec changes using rolling updates to avoid service disruption

### Key Entities

- **PortalExpose**: Custom resource defining which Kubernetes Service to expose, to which Portal relays, with what configuration. Contains spec (app name, service reference, relay targets, optional TunnelClass reference) and status (phase, public URL, tunnel pod counts, relay connection states, conditions).

- **TunnelClass**: Custom resource defining tunnel pod infrastructure configuration. Contains spec (replicas, size, optional nodeSelector, optional tolerations) and determines how tunnel pods are deployed (resource allocations, scheduling constraints).

- **Tunnel Deployment**: Kubernetes Deployment created by the controller for each PortalExpose. Runs the portal-tunnel container with configuration derived from PortalExpose and TunnelClass. Owned by the PortalExpose resource for automatic cleanup.

- **Service Reference**: Points to the Kubernetes Service being exposed (name and port). The tunnel pods forward traffic to this Service endpoint within the cluster.

- **Relay Target**: Specifies a Portal relay endpoint (name and WebSocket URL). Multiple targets can be specified for redundancy. Status tracks connection state per target.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can expose a Kubernetes Service to the internet by creating a single PortalExpose resource, with tunnel pods deployed automatically within 30 seconds
- **SC-002**: When a PortalExpose is deleted, all associated tunnel pods are removed automatically within 10 seconds
- **SC-003**: Status fields accurately reflect system state within 5 seconds of any state change (pod ready, relay connected, errors)
- **SC-004**: The controller handles 100+ PortalExpose resources in a single cluster without performance degradation
- **SC-005**: 95% of PortalExpose creations succeed on first attempt when Service exists and relays are reachable
- **SC-006**: Multi-relay configurations maintain service availability when up to 50% of relays are unavailable
- **SC-007**: TunnelClass changes propagate to existing PortalExposes within 60 seconds (reconciliation picks up changes)
- **SC-008**: Operators can diagnose exposure issues using only PortalExpose status fields and Kubernetes Events without needing to check pod logs
- **SC-009**: Controller recovers from crashes without leaving orphaned tunnel Deployments or inconsistent resource states
- **SC-010**: All errors include actionable messages that guide operators to resolution (e.g., "Service 'foo' not found in namespace 'default'")

## Clarifications

### Session 2025-01-18

- Q: When a TunnelClass is modified after PortalExposes are already using it, should the controller automatically update existing tunnel Deployments? → A: Auto-update but with rolling restart to avoid disruption (graceful update)
- Q: How does the controller determine the public URL to populate in status.publicURL? → A: Controller constructs URL from app name and relay domain (e.g., `https://{app.name}.{relay-domain}`)
- Q: When a PortalExpose has multiple tunnel pod replicas and some fail, what should the phase status show? → A: Phase shows "Degraded" when some but not all pods are ready (new phase for partial failures)
- Q: When some relays fail to connect (e.g., 2 of 3 relays connected), should this also trigger "Degraded" phase? → A: Yes - partial relay failures also trigger "Degraded" phase (consistent with pod failures)
- Q: When a PortalExpose is updated (e.g., relay targets changed), how should the controller apply these changes? → A: Rolling update - gradually replace tunnel pods with new configuration (zero downtime)

### Assumptions

- Portal relay endpoints are operated by external parties and follow the Portal relay protocol (WebSocket-based with end-to-end encryption)
- The portal-tunnel container image is available and maintained separately from this controller
- Kubernetes cluster has network connectivity to Portal relay endpoints (no firewall blocking outbound WSS connections)
- Users have appropriate RBAC permissions to create PortalExpose and TunnelClass resources
- Each app name (subdomain) should be unique per relay to avoid conflicts, though enforcement of this is handled by the Portal relay infrastructure, not the controller
- Default resource allocations (small/medium/large) are sufficient for most use cases, with production workloads using large sizing
- Tunnel pod health is determined by Kubernetes readiness/liveness probes configured in the portal-tunnel container
- Service port numbers are standard TCP ports (1-65535)
