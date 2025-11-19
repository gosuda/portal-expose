# Portal Expose Controller

A Kubernetes controller that automatically exposes your pods to the [Portal](https://gosuda.org/portal/) network, making local cluster services accessible on the public internet without managing servers or cloud infrastructure.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Usage](#usage)
  - [PortalExpose CRD](#portalexpose-crd)
  - [Ingress Support (Coming Soon)](#ingress-support-coming-soon)
- [Installation](#installation)
- [Configuration](#configuration)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

## Overview

Portal Expose Controller bridges Kubernetes and the Portal network, enabling you to expose cluster services to the internet through Portal's secure relay system. Simply create a `PortalExpose` resource, and the controller automatically deploys and manages tunnel pods that connect your services to Portal relays.

## Features

- **Declarative Configuration**: Define Portal exposures using Kubernetes-native CRDs
- **Automatic Tunnel Management**: Controller handles tunnel pod lifecycle automatically
- **Multi-Relay Support**: Connect to multiple Portal relays for redundancy
- **Service Discovery**: Automatically discovers and connects to Kubernetes Services
- **Status Reporting**: Real-time status updates on connection health and public URLs
- **Clean Lifecycle Management**: Automatic cleanup when PortalExpose resources are deleted

## Architecture

Portal Expose Controller uses a two-resource model: `TunnelClass` defines tunnel configuration, and `PortalExpose` specifies which services to expose.

![Architecture Diagram](docs/architecture.png)

**Learn more:** See [docs/architecture.md](docs/architecture.md) for detailed architecture documentation including components, data flow, design decisions, and security considerations.

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Portal relay endpoint (e.g., `wss://portal.gosuda.org/relay`)

### Install the Controller

```bash
# Install everything (CRDs, RBAC, controller, default TunnelClass)
kubectl apply -f https://raw.githubusercontent.com/gosuda/portal-expose/001-portal-controller/install.yaml
```

This installs:
- Custom Resource Definitions (PortalExpose, TunnelClass)
- RBAC permissions (ServiceAccount, ClusterRole, ClusterRoleBinding)
- Controller deployment (ghcr.io/gosuda/portal-expose-controller:0.1.0)
- Default TunnelClass for immediate use

### Expose Your First Service

```bash
# Create a sample app
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80

# Expose it through Portal
kubectl apply -f - <<EOF
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-nginx
spec:
  app:
    name: my-nginx
    service:
      name: nginx
      port: 80
  relay:
    targets:
    - name: relay1
      url: wss://portal.gosuda.org/relay
EOF

# Check status
kubectl get portalexpose my-nginx
kubectl get portalexpose my-nginx -o jsonpath='{.status.publicURL}'
```

Your app should now be accessible at the public URL shown in the status

## Usage

Portal Expose uses two main resources: `TunnelClass` for tunnel configuration and `PortalExpose` for service exposure.

### TunnelClass

`TunnelClass` defines the tunnel pod configuration. The controller manages all tunnel internals (image, encryption, timeouts) - you only choose performance tier and placement.

#### Basic Example

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: TunnelClass
metadata:
  name: default
  annotations:
    portal.gosuda.org/is-default-class: "true"
spec:
  replicas: 1
  size: small  # small, medium, large
```

See [examples/tunnel-class.yaml](examples/tunnel-class.yaml) for the default configuration.

#### Production Example

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: TunnelClass
metadata:
  name: production
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

See [examples/tunnel-class-production.yaml](examples/tunnel-class-production.yaml) for production setup.

#### TunnelClass Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `replicas` | int | Yes | Number of tunnel pod replicas |
| `size` | string | Yes | Performance tier: `small`, `medium`, or `large` |
| `nodeSelector` | map | No | Node selection constraints |
| `tolerations` | []object | No | Pod tolerations for node taints |

#### Size Reference

| Size | CPU Request | CPU Limit | Memory Request | Memory Limit | Use Case |
|------|-------------|-----------|----------------|--------------|----------|
| `small` | 100m | 500m | 128Mi | 512Mi | Development, low traffic |
| `medium` | 250m | 1000m | 256Mi | 1Gi | Production, moderate traffic |
| `large` | 500m | 2000m | 512Mi | 2Gi | High traffic, critical services |

**Note:** The controller controls tunnel image, encryption (always TLS), and connection settings. Users cannot customize these for security and consistency.

### PortalExpose CRD

`PortalExpose` defines how a Kubernetes service should be exposed through Portal.

#### Basic Example

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: hello-app
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

See [examples/basic-expose.yaml](examples/basic-expose.yaml) for a minimal example.

#### Production Example with Multiple Relays

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-app-portal
  namespace: production
spec:
  tunnelClassName: production
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
```

See [examples/portal-expose.yaml](examples/portal-expose.yaml) and [examples/multi-relay-expose.yaml](examples/multi-relay-expose.yaml) for more examples.

#### PortalExpose Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tunnelClassName` | string | No | TunnelClass to use (default: `default`) |
| `app.name` | string | Yes | Application name (becomes subdomain) |
| `app.service.name` | string | Yes | Kubernetes Service name to expose |
| `app.service.port` | int | Yes | Service port number |
| `relay.targets` | []object | Yes | List of Portal relay endpoints |
| `relay.targets[].name` | string | Yes | Relay identifier name |
| `relay.targets[].url` | string | Yes | WebSocket URL (wss://) |

#### Status Fields

The controller updates the status to reflect the current state:

```yaml
status:
  phase: Ready  # Pending, Ready, Failed
  publicURL: https://my-awesome-app.portal.gosuda.org
  tunnelPods:
    ready: 2
    total: 2
  relay:
    connected:
      - name: gosuda-portal
        status: Connected
        connectedAt: "2025-01-14T10:30:00Z"
      - name: thumbgo-portal
        status: Connected
        connectedAt: "2025-01-14T10:30:05Z"
  conditions:
    - type: TunnelDeployed
      status: "True"
      lastTransitionTime: "2025-01-14T10:30:00Z"
    - type: RelayConnected
      status: "True"
      message: "Connected to 2/2 relays"
```

### Examples

All example configurations are available in the [examples/](examples/) directory:

- **[basic-expose.yaml](examples/basic-expose.yaml)** - Minimal PortalExpose configuration
- **[portal-expose.yaml](examples/portal-expose.yaml)** - Production setup with multiple relays
- **[multi-relay-expose.yaml](examples/multi-relay-expose.yaml)** - Advanced multi-region relay setup
- **[tunnel-class.yaml](examples/tunnel-class.yaml)** - Default TunnelClass configuration
- **[tunnel-class-dev.yaml](examples/tunnel-class-dev.yaml)** - Development/minimal TunnelClass
- **[tunnel-class-production.yaml](examples/tunnel-class-production.yaml)** - Production TunnelClass with HA

See [examples/README.md](examples/README.md) for detailed usage instructions.

### Ingress Support (Coming Soon)

Future versions will support standard Kubernetes Ingress resources:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app-ingress
  annotations:
    portal.gosuda.org/relay-urls: "wss://portal.gosuda.org/relay,wss://portal.thumbgo.kr/relay"
spec:
  ingressClassName: portal-ingress
  rules:
  - host: my-app.portal.gosuda.org
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-app-service
            port:
              number: 8080
```

## Installation

### Quick Install (Recommended)

Install the controller with a single command:

```bash
kubectl apply -f https://raw.githubusercontent.com/gosuda/portal-expose/001-portal-controller/install.yaml
```

This installs:
- **CRDs**: PortalExpose and TunnelClass custom resources
- **RBAC**: ServiceAccount, ClusterRole, and ClusterRoleBinding
- **Controller**: Deployment running `ghcr.io/gosuda/portal-expose-controller:0.1.0`
- **Default TunnelClass**: Ready-to-use tunnel configuration

**Supported Platforms**: linux/amd64, linux/arm64

**Tunnel Image**: `ghcr.io/gosuda/portal-tunnel:1.0.0`

For detailed installation instructions, see [INSTALL.md](INSTALL.md).

### Verify Installation

```bash
# Check controller pod
kubectl get pods -n portal-expose-system

# Check CRDs
kubectl get crd | grep portal.gosuda.org

# Check default TunnelClass
kubectl get tunnelclass
```

### From Source

```bash
# Clone repository
git clone https://github.com/gosuda/portal-expose.git
cd portal-expose

# Install CRDs
make install

# Run controller locally (for development)
make run

# Or build and deploy to cluster
make docker-build docker-push IMG=your-registry/portal-expose:latest
make deploy IMG=your-registry/portal-expose:latest
```

### Using Helm (Coming Soon)

```bash
helm repo add portal-expose https://gosuda.github.io/portal-expose
helm install portal-expose portal-expose/portal-expose
```

## Configuration

### Controller Configuration

The controller can be configured via environment variables or command-line flags:

| Variable | Default | Description |
|----------|---------|-------------|
| `TUNNEL_IMAGE` | `ghcr.io/gosuda/portal-tunnel:1.0.0` | Tunnel container image (managed by controller) |
| `TUNNEL_VERSION` | `latest` | Tunnel image version tag |
| `DEFAULT_RELAY_URL` | `wss://portal.gosuda.org/relay` | Default relay if not specified |
| `METRICS_ADDR` | `:8080` | Metrics endpoint address |
| `HEALTH_PROBE_ADDR` | `:8081` | Health probe endpoint address |

**Note:** Tunnel image and version are controlled by the controller and cannot be overridden by users for security and consistency.

### RBAC Permissions

The controller requires the following permissions:

- `portalexposes`: all verbs (create, get, list, watch, update, delete)
- `tunnelclasses`: all verbs (create, get, list, watch, update, delete)
- `deployments`: create, get, list, watch, update, delete
- `services`: get, list, watch
- `events`: create, patch

## Development

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- kubebuilder (for CRD generation)

### Setup Development Environment

```bash
# Install dependencies
go mod download

# Generate CRDs and code
make generate manifests

# Run tests
make test

# Run controller locally
make run
```

### Project Structure

```
portal-expose/
├── api/
│   └── v1alpha1/
│       ├── portalexpose_types.go    # PortalExpose CRD definition
│       └── tunnelclass_types.go     # TunnelClass CRD definition
├── internal/
│   ├── controller/
│   │   ├── portalexpose_controller.go   # PortalExpose controller logic
│   │   └── tunnelclass_controller.go    # TunnelClass controller logic
│   └── tunnel/                          # Tunnel management logic
├── config/
│   ├── crd/                         # CRD manifests
│   ├── rbac/                        # RBAC configurations
│   └── manager/                     # Controller deployment
├── examples/                        # Example configurations
│   ├── README.md                    # Examples documentation
│   ├── basic-expose.yaml            # Minimal PortalExpose
│   ├── portal-expose.yaml           # Production PortalExpose
│   ├── multi-relay-expose.yaml      # Multi-relay setup
│   ├── tunnel-class.yaml            # Default TunnelClass
│   ├── tunnel-class-dev.yaml        # Development TunnelClass
│   └── tunnel-class-production.yaml # Production TunnelClass
└── main.go                          # Controller entry point
```

## Roadmap

### Phase 1: Core Functionality (Complete)
- [x] Project structure and design
- [x] CRD specifications (PortalExpose, TunnelClass)
- [x] Example configurations
- [x] TunnelClass CRD implementation
- [x] PortalExpose CRD implementation
- [x] Controller logic (TunnelClass & PortalExpose)
- [x] Tunnel deployment management
- [x] Status reporting
- [x] Multi-relay support
- [ ] E2E testing

### Phase 2: Advanced Features
- [x] Multi-relay failover (basic support)
- [ ] Automatic relay selection
- [ ] Metrics and monitoring (Prometheus)
- [ ] Helm chart
- [ ] Advanced tunnel scaling strategies

### Phase 3: Ingress Support
- [ ] Ingress controller implementation
- [ ] IngressClass registration
- [ ] Path-based routing
- [ ] TLS/certificate management

### Phase 4: Enterprise Features
- [ ] Authentication/authorization
- [ ] Rate limiting
- [ ] Access logs
- [ ] Dashboard UI
- [ ] Multi-tenancy support
