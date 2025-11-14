# Portal Expose Examples

This directory contains example configurations for using Portal Expose Controller.

## Quick Start

### 1. Apply a TunnelClass

Choose a tunnel configuration profile:

```bash
# Default - minimal resources (development)
kubectl apply -f tunnel-class.yaml

# Or production - high availability
kubectl apply -f tunnel-class-production.yaml

# Or dev - minimal for development
kubectl apply -f tunnel-class-dev.yaml
```

### 2. Expose Your Service

```bash
# Basic example
kubectl apply -f basic-expose.yaml

# Or production example with multiple relays
kubectl apply -f multi-relay-expose.yaml
```

### 3. Check Status

```bash
kubectl get portalexpose
kubectl describe portalexpose hello-app
```

## Files Overview

### TunnelClass Examples

| File | Use Case | Replicas | Size | Description |
|------|----------|----------|------|-------------|
| `tunnel-class.yaml` | Default | 1 | small | Default tunnel configuration |
| `tunnel-class-dev.yaml` | Development | 1 | small | Minimal resources for dev/test |
| `tunnel-class-production.yaml` | Production | 3 | large | High availability with node placement |

### PortalExpose Examples

| File | Complexity | Relays | Description |
|------|------------|--------|-------------|
| `basic-expose.yaml` | Simple | 1 | Minimal configuration |
| `portal-expose.yaml` | Medium | 2 | Production setup with custom namespace |
| `multi-relay-expose.yaml` | Advanced | 3 | Multi-region relay redundancy |

## TunnelClass Size Reference

The controller manages all tunnel internals (image, encryption, timeouts). You only choose the performance tier:

| Size | CPU Request | CPU Limit | Memory Request | Memory Limit | Use Case |
|------|-------------|-----------|----------------|--------------|----------|
| `small` | 100m | 500m | 128Mi | 512Mi | Development, low traffic |
| `medium` | 250m | 1000m | 256Mi | 1Gi | Production, moderate traffic |
| `large` | 500m | 2000m | 512Mi | 2Gi | High traffic, critical services |

## What the Controller Manages

Users **cannot customize** these (enforced by controller):

- ✋ Tunnel container image
- ✋ Encryption settings (always TLS)
- ✋ Connection timeouts and keepalive
- ✋ Security context
- ✋ Container ports and protocols

Users **can customize**:

- ✅ Performance tier (`size`: small/medium/large)
- ✅ Replica count
- ✅ Node placement (nodeSelector, tolerations)
- ✅ Relay endpoints
- ✅ Application name and service target

## Common Workflows

### Development Setup

```bash
# 1. Create dev tunnel class
kubectl apply -f tunnel-class-dev.yaml

# 2. Expose your app
cat <<EOF | kubectl apply -f -
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-dev-app
spec:
  tunnelClassName: dev
  app:
    name: my-dev-app
    service:
      name: my-service
      port: 8080
  relay:
    targets:
      - name: default
        url: wss://portal.gosuda.org/relay
EOF

# 3. Get public URL
kubectl get portalexpose my-dev-app -o jsonpath='{.status.publicURL}'
```

### Production Setup

```bash
# 1. Create production tunnel class
kubectl apply -f tunnel-class-production.yaml

# 2. Expose with multiple relays
kubectl apply -f multi-relay-expose.yaml

# 3. Monitor status
kubectl get portalexpose critical-app -w
```

### Update Tunnel Configuration

```bash
# Scale up replicas
kubectl patch tunnelclass production -p '{"spec":{"replicas":5}}'

# Change size tier
kubectl patch tunnelclass production -p '{"spec":{"size":"large"}}'
```

## Expected Status Output

After applying a PortalExpose, check status:

```bash
kubectl get portalexpose hello-app -o yaml
```

Expected status:

```yaml
status:
  phase: Ready  # Pending, Ready, Failed
  publicURL: https://hello-app.portal.gosuda.org
  tunnelPods:
    ready: 1
    total: 1
  relay:
    connected:
      - name: default
        status: Connected
        connectedAt: "2025-01-14T10:30:00Z"
  conditions:
    - type: TunnelDeployed
      status: "True"
      lastTransitionTime: "2025-01-14T10:30:00Z"
    - type: RelayConnected
      status: "True"
      message: "Connected to 1/1 relays"
```

## Troubleshooting

### Check tunnel pods

```bash
kubectl get pods -l app=portal-tunnel
kubectl logs -l app=portal-tunnel
```

### Check PortalExpose events

```bash
kubectl describe portalexpose hello-app
```

### Verify service exists

```bash
kubectl get service hello-app
```

## Next Steps

1. Read the main [README](../README.md) for architecture details
2. Check [Configuration](../README.md#configuration) for controller setup
3. See [Development](../README.md#development) to contribute
