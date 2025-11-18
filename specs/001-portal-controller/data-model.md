# Data Model: Portal Expose Kubernetes Controller

**Feature**: Portal Expose Kubernetes Controller
**Date**: 2025-01-18
**Purpose**: Define CRD structures, status fields, and entity relationships

## API Group and Version

- **Group**: `portal.gosuda.org`
- **Version**: `v1alpha1`
- **Kind**: `PortalExpose`, `TunnelClass`

## Entity: PortalExpose

### Overview

Custom resource defining which Kubernetes Service to expose through Portal relays.

### Spec Fields

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `app.name` | string | Yes | DNS-1123 label | Application name, becomes subdomain (e.g., "my-app" → "my-app.portal.gosuda.org") |
| `app.service.name` | string | Yes | Valid K8s name | Kubernetes Service name in same namespace |
| `app.service.port` | int32 | Yes | 1-65535 | Service port number to expose |
| `relay.targets` | []RelayTarget | Yes | Min 1 target | List of Portal relay endpoints |
| `tunnelClassName` | string | No | Valid K8s name | TunnelClass reference (uses default if omitted) |

#### RelayTarget

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `name` | string | Yes | Unique within array | Relay identifier (e.g., "gosuda-portal") |
| `url` | string | Yes | WSS URL format | WebSocket relay URL (e.g., "wss://portal.gosuda.org/relay") |

### Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current state: Pending \| Ready \| Degraded \| Failed |
| `publicURL` | string | Accessible endpoint (e.g., "https://my-app.portal.gosuda.org") |
| `tunnelPods.ready` | int32 | Number of ready tunnel pods |
| `tunnelPods.total` | int32 | Desired number of tunnel pods |
| `relay.connected` | []RelayConnectionStatus | Per-relay connection state |
| `conditions` | []metav1.Condition | Detailed status conditions |

#### RelayConnectionStatus

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Relay name (matches spec.relay.targets[].name) |
| `status` | string | Connected \| Disconnected \| Unknown |
| `connectedAt` | *metav1.Time | Timestamp when connection established (nil if not connected) |
| `lastError` | string | Last connection error message (empty if no error) |

#### Conditions

Standard condition types (type field in metav1.Condition):

- `Available`: PortalExpose is Ready or Degraded (True), vs Failed/Pending (False)
- `Progressing`: Tunnel Deployment rollout in progress (True) vs stable (False)
- `TunnelDeploymentReady`: All tunnel pods ready (True) vs partial/none (False)
- `RelayConnected`: All relays connected (True) vs partial/none (False)
- `ServiceExists`: Referenced Service found (True) vs not found (False)

### Example YAML

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-app-portal
  namespace: default
spec:
  app:
    name: my-awesome-app
    service:
      name: my-app-service
      port: 8080
  relay:
    targets:
      - name: gosuda-portal
        url: wss://portal.gosuda.org/relay
      - name: thumbgo-portal
        url: wss://portal.thumbgo.kr/relay
  tunnelClassName: production
status:
  phase: Ready
  publicURL: https://my-awesome-app.portal.gosuda.org
  tunnelPods:
    ready: 3
    total: 3
  relay:
    connected:
      - name: gosuda-portal
        status: Connected
        connectedAt: "2025-01-18T10:30:00Z"
      - name: thumbgo-portal
        status: Connected
        connectedAt: "2025-01-18T10:30:05Z"
  conditions:
    - type: Available
      status: "True"
      reason: AllComponentsReady
      message: All tunnel pods and relays are healthy
      lastTransitionTime: "2025-01-18T10:30:05Z"
    - type: TunnelDeploymentReady
      status: "True"
      reason: AllPodsReady
      message: 3/3 tunnel pods ready
      lastTransitionTime: "2025-01-18T10:30:00Z"
    - type: RelayConnected
      status: "True"
      reason: AllRelaysConnected
      message: Connected to 2/2 relays
      lastTransitionTime: "2025-01-18T10:30:05Z"
```

## Entity: TunnelClass

### Overview

Custom resource defining tunnel pod infrastructure configuration (how tunnels run).

### Spec Fields

| Field | Type | Required | Validation | Description |
|-------|------|----------|------------|-------------|
| `replicas` | int32 | Yes | ≥ 1 | Number of tunnel pod replicas |
| `size` | string | Yes | Enum: small \| medium \| large | Resource allocation tier |
| `nodeSelector` | map[string]string | No | Valid labels | Node selection constraints |
| `tolerations` | []corev1.Toleration | No | Valid tolerations | Pod tolerations for taints |

#### Size Resource Mappings

| Size | CPU Request | CPU Limit | Memory Request | Memory Limit |
|------|-------------|-----------|----------------|--------------|
| small | 100m | 500m | 128Mi | 512Mi |
| medium | 250m | 1000m | 256Mi | 1Gi |
| large | 500m | 2000m | 512Mi | 2Gi |

### Status Fields

TunnelClass status is read-only and informational (controllers watch TunnelClass but don't update its status).

| Field | Type | Description |
|-------|------|-------------|
| `observedGeneration` | int64 | Last observed spec generation (for change detection) |
| `conditions` | []metav1.Condition | Health conditions |

### Annotations

- `portal.gosuda.org/is-default-class: "true"`: Marks this TunnelClass as default (used when PortalExpose doesn't specify tunnelClassName)

### Example YAML

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: TunnelClass
metadata:
  name: production
  annotations:
    portal.gosuda.org/is-default-class: "true"
spec:
  replicas: 3
  size: large
  nodeSelector:
    workload-type: tunnel
  tolerations:
    - key: tunnel
      operator: Equal
      value: "true"
      effect: NoSchedule
```

## Entity: Tunnel Deployment (Managed Resource)

### Overview

Kubernetes Deployment created by the controller for each PortalExpose. Not a CRD, but a standard Deployment resource.

### Naming Convention

- Name: `{portalexpose-name}-tunnel`
- Namespace: Same as PortalExpose
- Example: PortalExpose "my-app-portal" → Deployment "my-app-portal-tunnel"

### Labels

| Label | Value | Purpose |
|-------|-------|---------|
| `app.kubernetes.io/name` | `portal-tunnel` | Standard app label |
| `app.kubernetes.io/component` | `tunnel` | Component type |
| `app.kubernetes.io/managed-by` | `portal-expose-controller` | Controller identity |
| `portal.gosuda.org/portalexpose` | `{portalexpose-name}` | Link to owning PortalExpose |

### Owner References

- Set PortalExpose as owner (enables automatic garbage collection)
- BlockOwnerDeletion: true (prevents deletion if PortalExpose has finalizer)

### Pod Template Spec

Derived from PortalExpose and TunnelClass:

- **Image**: Controlled by controller (e.g., `ghcr.io/gosuda/portal-tunnel:v0.1.0`)
- **Args**: Tunnel configuration (relay URLs, service name/port, app name)
- **Resources**: From TunnelClass.spec.size mapping
- **NodeSelector**: From TunnelClass.spec.nodeSelector
- **Tolerations**: From TunnelClass.spec.tolerations
- **Replicas**: From TunnelClass.spec.replicas

### Update Strategy

- Type: RollingUpdate
- MaxSurge: 25%
- MaxUnavailable: 25%

Ensures zero-downtime updates when TunnelClass or PortalExpose changes.

## Relationships

```
PortalExpose
  ├─ references → Service (read-only, validates existence)
  ├─ references → TunnelClass (read-only, inherits config)
  └─ owns → Deployment (manages lifecycle)
       └─ creates → Pods (Kubernetes Deployment controller manages)

TunnelClass
  └─ referenced by → PortalExposes (many-to-one)
```

## State Transitions

### PortalExpose Phase Transitions

```
                     ┌─────────┐
                     │ Pending │ (Initial state)
                     └────┬────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
    ┌────────┐      ┌─────────┐      ┌────────┐
    │ Failed │◄────►│Degraded │◄────►│ Ready  │
    └────────┘      └─────────┘      └────────┘
         │                │                │
         └────────────────┴────────────────┘
                          │
                          ▼
                  (Spec update resets
                   to Pending)
```

**Transition Triggers**:
- Pending → Ready: All pods ready AND all relays connected
- Pending → Degraded: Some pods ready OR some relays connected
- Pending → Failed: No pods ready OR no relays connected OR Service missing
- Ready → Degraded: Pod failure OR relay disconnection (partial)
- Ready → Failed: All pods fail OR all relays fail
- Degraded → Ready: All pods/relays recovered
- Degraded → Failed: All remaining pods/relays fail
- Failed → Pending: PortalExpose spec updated (trigger re-reconciliation)

## Validation Rules

### PortalExpose Validation

1. **app.name**: Must be valid DNS-1123 label (lowercase alphanumeric + hyphens, max 63 chars)
2. **app.service.port**: Must be 1-65535
3. **relay.targets**: Must have at least 1 entry
4. **relay.targets[].name**: Must be unique within array
5. **relay.targets[].url**: Must match pattern `wss://[domain]/[path]`
6. **tunnelClassName**: If specified, referenced TunnelClass must exist (validated at reconcile time, not admission)

### TunnelClass Validation

1. **replicas**: Must be ≥ 1
2. **size**: Must be exactly "small", "medium", or "large"
3. **nodeSelector**: Keys/values must be valid Kubernetes labels
4. **tolerations**: Must follow corev1.Toleration schema

## Field Mappings: PortalExpose + TunnelClass → Tunnel Deployment

| Deployment Field | Source | Value |
|------------------|--------|-------|
| metadata.name | PortalExpose | `{portalexpose.metadata.name}-tunnel` |
| metadata.namespace | PortalExpose | `{portalexpose.metadata.namespace}` |
| spec.replicas | TunnelClass | `{tunnelclass.spec.replicas}` |
| spec.template.spec.nodeSelector | TunnelClass | `{tunnelclass.spec.nodeSelector}` |
| spec.template.spec.tolerations | TunnelClass | `{tunnelclass.spec.tolerations}` |
| spec.template.spec.containers[0].resources | TunnelClass | Size mapping (small/medium/large) |
| spec.template.spec.containers[0].image | Controller Config | `ghcr.io/gosuda/portal-tunnel:{version}` |
| spec.template.spec.containers[0].args | PortalExpose | Relay URLs, service name/port, app name |

## Indexing for Efficient Queries

Controller should index:

1. **PortalExposes by TunnelClass**: To efficiently find all PortalExposes using a TunnelClass when it changes
2. **PortalExposes by Service**: To detect when a referenced Service is deleted
3. **Deployments by PortalExpose**: To correlate Deployment events back to owning PortalExpose

Implemented via controller-runtime field indexers.
