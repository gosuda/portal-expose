# Reconciliation Contract: Portal Expose Controller

**Feature**: Portal Expose Kubernetes Controller
**Date**: 2025-01-18
**Purpose**: Define controller reconciliation behavior and contracts

## Overview

Kubernetes controllers don't expose REST/GraphQL APIs in the traditional sense. Instead, they implement reconciliation loops that watch CRD resources and reconcile them to desired state. This document defines the "contract" for how the controller behaves when resources change.

## PortalExpose Controller Reconciliation Contract

### Input

Reconciliation is triggered when:
1. PortalExpose resource is created/updated/deleted
2. TunnelClass referenced by PortalExpose is updated
3. Tunnel Deployment owned by PortalExpose changes (pod failures, external modifications)
4. Periodic resync (default every 10 hours, configurable)

### Reconciliation Logic

```
FUNCTION Reconcile(portalExpose PortalExpose) -> (Result, Error)

  // 1. Handle deletion
  IF portalExpose.DeletionTimestamp IS SET:
    DELETE tunnel Deployment
    IF Deployment still exists:
      RETURN (Requeue=true, nil) // Wait for Deployment deletion
    REMOVE finalizer from portalExpose
    RETURN (Done, nil)

  // 2. Add finalizer if missing
  IF finalizer NOT present:
    ADD finalizer "portal.gosuda.org/cleanup-tunnel-deployment"
    UPDATE portalExpose
    RETURN (Done, nil)

  // 3. Validate referenced Service exists
  service := GET Service(portalExpose.spec.app.service.name)
  IF service NOT FOUND:
    SET status.phase = "Failed"
    SET condition ServiceExists = False ("Service not found")
    UPDATE status
    RETURN (Done, nil) // Don't requeue, wait for Service creation event

  // 4. Resolve TunnelClass
  tunnelClass := GET TunnelClass(portalExpose.spec.tunnelClassName)
  IF tunnelClassName IS EMPTY:
    tunnelClass = GET default TunnelClass (annotation portal.gosuda.org/is-default-class=true)
  IF tunnelClass NOT FOUND:
    SET status.phase = "Failed"
    SET condition TunnelClassExists = False ("TunnelClass not found")
    UPDATE status
    RETURN (Done, nil)

  // 5. Generate desired Deployment spec
  desiredDeployment := BuildDeployment(portalExpose, tunnelClass)

  // 6. Reconcile Deployment
  existingDeployment := GET Deployment(portalExpose.name + "-tunnel")
  IF existingDeployment NOT FOUND:
    CREATE desiredDeployment
    SET status.phase = "Pending"
    UPDATE status
    RETURN (Requeue=true, nil) // Requeue to check pod readiness

  IF existingDeployment.Spec != desiredDeployment.Spec:
    UPDATE existingDeployment with desiredDeployment.Spec
    SET condition Progressing = True ("Rolling update in progress")
    UPDATE status
    RETURN (Requeue=true, nil)

  // 7. Compute status from Deployment and tunnel pods
  pods := LIST Pods(labels matching Deployment)
  readyPods := COUNT pods WHERE pod.Status.Conditions[Ready] == True
  totalPods := tunnelClass.spec.replicas

  SET status.tunnelPods.ready = readyPods
  SET status.tunnelPods.total = totalPods

  // 8. Determine relay connection status (simulated - actual status from tunnel pod annotations/logs)
  // NOTE: In real implementation, tunnel pods report relay status via annotations or metrics
  relayStatuses := []
  FOR EACH relay IN portalExpose.spec.relay.targets:
    // Simplified: Assume Connected if pods are ready (real implementation checks tunnel pod status)
    IF readyPods > 0:
      relayStatus = RelayConnectionStatus{
        name: relay.name,
        status: "Connected",
        connectedAt: NOW(),
      }
    ELSE:
      relayStatus = RelayConnectionStatus{
        name: relay.name,
        status: "Disconnected",
      }
    relayStatuses.append(relayStatus)

  SET status.relay.connected = relayStatuses

  // 9. Compute phase
  allPodsReady := (readyPods == totalPods)
  somePodsReady := (readyPods > 0)
  allRelaysConnected := ALL relayStatuses have status="Connected"
  someRelaysConnected := ANY relayStatuses have status="Connected"

  IF allPodsReady AND allRelaysConnected:
    SET status.phase = "Ready"
  ELSE IF somePodsReady OR someRelaysConnected:
    SET status.phase = "Degraded"
  ELSE:
    SET status.phase = "Failed"

  // 10. Construct public URL
  primaryRelay := portalExpose.spec.relay.targets[0]
  relayDomain := EXTRACT_DOMAIN(primaryRelay.url) // "wss://portal.gosuda.org/relay" -> "portal.gosuda.org"
  SET status.publicURL = "https://" + portalExpose.spec.app.name + "." + relayDomain

  // 11. Update conditions
  SET condition TunnelDeploymentReady = (allPodsReady ? True : False)
  SET condition RelayConnected = (allRelaysConnected ? True : False)
  SET condition ServiceExists = True
  SET condition Available = (status.phase IN [Ready, Degraded] ? True : False)
  SET condition Progressing = (existingDeployment.Status.UpdatedReplicas < totalPods ? True : False)

  // 12. Update status
  UPDATE portalExpose.status

  // 13. Emit Kubernetes Event
  IF phase changed:
    EMIT Event(type=Normal/Warning, reason=phase, message="PortalExpose is {phase}")

  RETURN (Done, nil)
END FUNCTION
```

### Output (Status Updates)

Status fields updated by controller:
- `status.phase`: Pending | Ready | Degraded | Failed
- `status.publicURL`: Constructed public endpoint
- `status.tunnelPods.ready` / `status.tunnelPods.total`: Pod counts
- `status.relay.connected[]`: Per-relay connection status
- `status.conditions[]`: Detailed condition array

### Error Handling

| Error Condition | Phase | Requeue | Backoff |
|-----------------|-------|---------|---------|
| Service not found | Failed | No | Wait for Service creation watch event |
| TunnelClass not found | Failed | No | Wait for TunnelClass creation |
| Deployment creation failed | Pending | Yes | Exponential backoff (5ms → 1000s) |
| Tunnel pods not ready | Pending/Degraded | Yes | Every 30s (status check interval) |
| Relay connection failed | Degraded/Failed | Yes | Exponential backoff |
| Deployment update failed | Current phase | Yes | Exponential backoff |

### Performance Guarantees

- **Reconciliation latency**: <500ms (excluding Kubernetes API calls)
- **Status update latency**: <5 seconds from state change to status update
- **Deployment creation**: <30 seconds from PortalExpose creation to pods starting
- **Rolling update time**: Depends on Deployment rolling update parameters (typically 30-60s for 3 replicas)

## TunnelClass Controller Reconciliation Contract

### Input

Reconciliation is triggered when:
1. TunnelClass resource is created/updated/deleted
2. Periodic resync

### Reconciliation Logic

```
FUNCTION Reconcile(tunnelClass TunnelClass) -> (Result, Error)

  // TunnelClass itself doesn't manage resources, but changes trigger PortalExpose updates

  // 1. Find all PortalExposes referencing this TunnelClass
  portalExposes := LIST PortalExposes WHERE spec.tunnelClassName == tunnelClass.name

  // 2. Enqueue each PortalExpose for reconciliation
  FOR EACH portalExpose IN portalExposes:
    ENQUEUE portalExpose for reconciliation

  // 3. Update observedGeneration (for tracking spec changes)
  SET status.observedGeneration = tunnelClass.metadata.generation
  UPDATE status

  RETURN (Done, nil)
END FUNCTION
```

### Output

- Enqueues affected PortalExposes for reconciliation (triggers rolling updates)
- Updates `status.observedGeneration` to track TunnelClass changes

## Deployment Watch Contract

The controller also watches Deployment resources (owned by PortalExposes).

### Input

Triggered when:
1. Deployment is modified (externally or by Deployment controller)
2. Deployment is deleted (unexpectedly, not by controller)

### Reconciliation Logic

```
FUNCTION OnDeploymentChange(deployment Deployment) -> (Result, Error)

  // Find owning PortalExpose via owner reference
  portalExpose := GET owner from deployment.metadata.ownerReferences

  IF portalExpose NOT FOUND:
    // Deployment not owned by PortalExpose, ignore
    RETURN (Done, nil)

  // Enqueue PortalExpose for reconciliation (will sync status)
  ENQUEUE portalExpose for reconciliation

  RETURN (Done, nil)
END FUNCTION
```

## Service Watch Contract

The controller watches Service resources to detect when a referenced Service is created or deleted.

### Input

Triggered when:
1. Service is created (may resolve PortalExpose waiting for Service)
2. Service is deleted (should mark PortalExpose as Failed)

### Reconciliation Logic

```
FUNCTION OnServiceChange(service Service) -> (Result, Error)

  // Find all PortalExposes referencing this Service
  portalExposes := LIST PortalExposes WHERE spec.app.service.name == service.name AND namespace == service.namespace

  // Enqueue each for reconciliation
  FOR EACH portalExpose IN portalExposes:
    ENQUEUE portalExpose for reconciliation

  RETURN (Done, nil)
END FUNCTION
```

## Webhook Contracts (Optional Future Enhancement)

Admission webhooks can validate PortalExpose/TunnelClass at creation/update time (currently validation happens at reconcile time).

### Validating Webhook: PortalExpose

**Endpoint**: `POST /validate-portal-gosuda-org-v1alpha1-portalexpose`

**Input**: AdmissionReview (contains PortalExpose object)

**Validation Rules**:
1. `spec.app.name` matches DNS-1123 label format
2. `spec.app.service.port` in range 1-65535
3. `spec.relay.targets` has at least 1 entry
4. `spec.relay.targets[].name` values are unique
5. `spec.relay.targets[].url` matches `wss://.*` pattern

**Output**: AdmissionReview with allowed=true/false

### Validating Webhook: TunnelClass

**Endpoint**: `POST /validate-portal-gosuda-org-v1alpha1-tunnelclass`

**Validation Rules**:
1. `spec.replicas` ≥ 1
2. `spec.size` in ["small", "medium", "large"]

## Events Emitted

| Event Type | Reason | Message | Trigger |
|------------|--------|---------|---------|
| Normal | Created | PortalExpose created, deploying tunnel pods | PortalExpose created |
| Normal | Ready | All tunnel pods and relays are healthy | Phase → Ready |
| Warning | Degraded | Partial failure: {details} | Phase → Degraded |
| Warning | ServiceNotFound | Referenced Service '{name}' not found | Service validation failed |
| Warning | TunnelClassNotFound | Referenced TunnelClass '{name}' not found | TunnelClass lookup failed |
| Error | Failed | All tunnel pods or relays unavailable | Phase → Failed |
| Normal | Updated | Configuration updated, rolling update in progress | TunnelClass changed |
| Normal | Deleted | PortalExpose deleted, tunnel pods cleaned up | PortalExpose deleted |

## Metrics Exposed (Future)

Prometheus metrics endpoint (`:8080/metrics`):

- `portalexpose_reconcile_duration_seconds`: Histogram of reconciliation duration
- `portalexpose_reconcile_total`: Counter of reconciliations by result (success/error)
- `portalexpose_status_phase`: Gauge of PortalExposes by phase (Ready/Degraded/Failed/Pending)
- `portalexpose_relay_connections_total`: Gauge of relay connections by status (Connected/Disconnected)
- `tunnelclass_referenced_total`: Gauge of PortalExposes per TunnelClass
