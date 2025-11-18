# Portal Expose Examples Index

This directory contains comprehensive examples for Portal Expose controller usage.

## Quick Start

**[quick-start.yaml](quick-start.yaml)** - Complete end-to-end example
The fastest way to get started. Includes everything needed:
- Default TunnelClass
- Sample nginx deployment
- Service configuration
- PortalExpose resource

```bash
kubectl apply -f examples/quick-start.yaml
kubectl get portalexpose hello-world -o jsonpath='{.status.publicURL}'
```

## TunnelClass Examples

TunnelClass defines tunnel pod configuration (replicas, resources, placement).

### [tunnel-class.yaml](tunnel-class.yaml) - Default Configuration
Basic TunnelClass marked as default. Used when PortalExpose doesn't specify `tunnelClassName`.

```yaml
spec:
  replicas: 1
  size: small  # 100m CPU, 128Mi RAM
```

### [tunnel-class-dev.yaml](tunnel-class-dev.yaml) - Development
Minimal resources for testing and development.

```yaml
spec:
  replicas: 1
  size: small
```

### [tunnel-class-production.yaml](tunnel-class-production.yaml) - Production
High availability with node placement controls.

```yaml
spec:
  replicas: 3
  size: large  # 500m CPU, 512Mi RAM
  nodeSelector:
    workload-type: tunnel
  tolerations:
    - key: tunnel
      operator: Equal
      value: "true"
```

## PortalExpose Examples

PortalExpose defines which services to expose and relay configuration.

### [basic-expose.yaml](basic-expose.yaml) - Minimal Example
Simplest possible configuration using default TunnelClass.

```yaml
spec:
  app:
    name: hello-app
    service:
      name: hello-app
      port: 8080
  relay:
    targets:
      - name: default
        url: wss://portal.gosuda.org/relay
```

**Public URL:** `https://hello-app.portal.gosuda.org`

### [portal-expose.yaml](portal-expose.yaml) - Standard Configuration
Single service with basic setup.

### [multi-relay-expose.yaml](multi-relay-expose.yaml) - Redundancy
Multiple relays for high availability and geo-distribution.

```yaml
spec:
  relay:
    targets:
      - name: primary-relay
        url: wss://portal.gosuda.org/relay
      - name: backup-relay
        url: wss://portal.thumbgo.kr/relay
      - name: eu-relay
        url: wss://portal-eu.gosuda.org/relay
```

## Complete Setup Examples

### [development-setup.yaml](development-setup.yaml) - Dev Environment
Complete setup for development/testing:
- Dev TunnelClass (1 replica, small)
- Single relay
- Minimal resources

**Use case:** Local development, testing, cost optimization

### [production-setup.yaml](production-setup.yaml) - Production Environment
Complete production deployment:
- Production TunnelClass (3 replicas, large)
- Multi-relay redundancy (US, EU, Asia)
- Node selectors and tolerations
- High availability

**Use case:** Production workloads, mission-critical services

### [microservices-setup.yaml](microservices-setup.yaml) - Microservices
Multiple services sharing a TunnelClass:
- Shared medium TunnelClass
- 4 different microservices exposed
- Different relay configurations per service

**Use case:** Microservices architecture, multiple services

**Public URLs:**
- `https://api.portal.gosuda.org` - API Gateway
- `https://users.portal.gosuda.org` - User Service
- `https://orders.portal.gosuda.org` - Order Service
- `https://ws.portal.gosuda.org` - WebSocket Service

## Common Patterns

### Pattern 1: Default TunnelClass
Most common pattern - use default TunnelClass for simplicity:

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-app
spec:
  # tunnelClassName not specified - uses default
  app:
    name: my-app
    service:
      name: my-app
      port: 8080
  relay:
    targets:
      - name: primary
        url: wss://portal.gosuda.org/relay
```

### Pattern 2: Explicit TunnelClass
Specify TunnelClass for different resource profiles:

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: critical-app
spec:
  tunnelClassName: production  # Explicit class
  app:
    name: critical-app
    service:
      name: api-service
      port: 443
  relay:
    targets:
      - name: primary
        url: wss://portal.gosuda.org/relay
```

### Pattern 3: Multi-Relay Redundancy
Multiple relays for failover and geo-distribution:

```yaml
spec:
  relay:
    targets:
      - name: us-primary
        url: wss://portal.gosuda.org/relay
      - name: eu-backup
        url: wss://portal-eu.gosuda.org/relay
      - name: asia-backup
        url: wss://portal-asia.gosuda.org/relay
```

## TunnelClass Size Reference

| Size | CPU Request | CPU Limit | Memory Request | Memory Limit | Use Case |
|------|-------------|-----------|----------------|--------------|----------|
| `small` | 100m | 500m | 128Mi | 512Mi | Development, low traffic |
| `medium` | 250m | 1000m | 256Mi | 1Gi | Production, moderate traffic |
| `large` | 500m | 2000m | 512Mi | 2Gi | High traffic, critical services |

## Status Phases

After creating a PortalExpose, check its status:

```bash
kubectl get portalexpose <name> -o yaml
```

**Phases:**
- `Pending` - Tunnel pods starting, relays connecting
- `Ready` - All pods ready, all relays connected
- `Degraded` - Some pods/relays unavailable
- `Failed` - Service not found or critical error

**Example status:**
```yaml
status:
  phase: Ready
  publicURL: https://my-app.portal.gosuda.org
  tunnelPods:
    ready: 2
    total: 2
  relay:
    connected:
      - name: primary
        status: Connected
        connectedAt: "2025-01-19T10:00:00Z"
  conditions:
    - type: ServiceExists
      status: "True"
    - type: TunnelDeploymentReady
      status: "True"
    - type: RelayConnected
      status: "True"
    - type: Available
      status: "True"
```

## Troubleshooting

### Check PortalExpose status
```bash
kubectl get portalexpose
kubectl describe portalexpose <name>
```

### Check tunnel pods
```bash
kubectl get pods -l portal.gosuda.org/portalexpose=<name>
kubectl logs -l portal.gosuda.org/portalexpose=<name>
```

### Check events
```bash
kubectl get events --sort-by='.lastTimestamp'
```

### Common issues

**Service not found:**
```yaml
status:
  phase: Failed
  conditions:
    - type: ServiceExists
      status: "False"
      message: "Service 'my-app' not found"
```
**Solution:** Ensure the Service exists in the same namespace.

**No default TunnelClass:**
```yaml
status:
  phase: Failed
  conditions:
    - type: TunnelClassExists
      status: "False"
      message: "No default TunnelClass found"
```
**Solution:** Create a TunnelClass with `portal.gosuda.org/is-default-class: "true"`.

## Next Steps

1. **Quick test:** Apply [quick-start.yaml](quick-start.yaml)
2. **Development:** Use [development-setup.yaml](development-setup.yaml)
3. **Production:** Adapt [production-setup.yaml](production-setup.yaml)
4. **Multiple services:** See [microservices-setup.yaml](microservices-setup.yaml)

For more information, see the main [README.md](../README.md).
