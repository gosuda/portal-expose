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

```
┌─────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                 │
│                                                     │
│  ┌──────────────┐      ┌─────────────────────┐      │
│  │ PortalExpose │─────▶│ Expose Controller   │      │
│  │   (CRD)      │      │   (watches CRDs)    │      │
│  └──────────────┘      └──────────┬──────────┘      │
│                                   │                 │
│                                   ▼                 │
│  ┌──────────────┐      ┌─────────────────────┐      │
│  │ Your Service │─────▶│ Tunnel Deployment   │      │
│  │ (app pods)   │      │   (portal-tunnel)   │      │
│  └──────────────┘      └─────────────────────┘      │
└─────────────────────────────────────────────────────┘
                                   │
                                   │ WebSocket (WSS)
                                   ▼
                        ┌─────────────────────┐
                        │   Portal Relay      │
                        │  (gosuda.org, etc)  │
                        └──────────┬──────────┘
                                   │
                                   ▼
                            Public Internet
                     (https://myapp.portal.gosuda.org)
```

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured
- Portal relay endpoint (e.g., `wss://portal.gosuda.org/relay`)

### Install the Controller

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/YOUR_ORG/portal-expose-controller/main/config/crd/bases/portal.gosuda.org_portalexposes.yaml

# Install controller
kubectl apply -f https://raw.githubusercontent.com/YOUR_ORG/portal-expose-controller/main/config/deploy/controller.yaml
```

### Expose Your First Service

```bash
# Create a sample app
kubectl create deployment hello-app --image=gcr.io/google-samples/hello-app:1.0
kubectl expose deployment hello-app --port=8080

# Expose it through Portal
cat <<EOF | kubectl apply -f -
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: hello-app-portal
spec:
  serviceName: hello-app
  servicePort: 8080
  portalName: my-hello-app
  relayURLs:
    - wss://portal.gosuda.org/relay
EOF

# Check status
kubectl get portalexpose hello-app-portal
```

Your app should now be accessible at `https://my-hello-app.portal.gosuda.org`

## Usage

### PortalExpose CRD

The `PortalExpose` custom resource defines how a Kubernetes service should be exposed through Portal.

#### Basic Example

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-app-portal
  namespace: default
spec:
  # Required: Service to expose
  serviceName: my-app-service
  servicePort: 8080

  # Required: Portal subdomain name
  portalName: my-awesome-app

  # Required: Portal relay endpoints
  relayURLs:
    - wss://portal.gosuda.org/relay
```

#### Advanced Example

```yaml
apiVersion: portal.gosuda.org/v1alpha1
kind: PortalExpose
metadata:
  name: my-app-portal
  namespace: production
spec:
  serviceName: my-app-service
  servicePort: 8080
  portalName: my-awesome-app

  # Multiple relays for redundancy
  relayURLs:
    - wss://portal.gosuda.org/relay
    - wss://portal.thumbgo.kr/relay

  # Tunnel configuration
  tunnel:
    replicas: 2  # Run multiple tunnel instances
    image: portal-tunnel:v1.0.0  # Custom tunnel image
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 500m
        memory: 512Mi

  # Encryption settings
  encryption:
    enabled: true
    protocol: tls
```

#### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `serviceName` | string | Yes | Name of the Kubernetes Service to expose |
| `servicePort` | int | Yes | Port number of the service |
| `portalName` | string | Yes | Subdomain name on Portal (e.g., "myapp" → myapp.portal.gosuda.org) |
| `relayURLs` | []string | Yes | List of Portal relay WebSocket URLs |
| `tunnel.replicas` | int | No | Number of tunnel pod replicas (default: 1) |
| `tunnel.image` | string | No | Custom portal-tunnel image |
| `tunnel.resources` | object | No | Resource requests/limits for tunnel pods |
| `encryption.enabled` | bool | No | Enable end-to-end encryption (default: true) |

#### Status Fields

The controller updates the status to reflect the current state:

```yaml
status:
  phase: Ready  # Pending, Ready, Failed
  publicURL: https://my-awesome-app.portal.gosuda.org
  tunnelPods:
    ready: 2
    total: 2
  conditions:
    - type: TunnelDeployed
      status: "True"
      lastTransitionTime: "2025-01-13T10:30:00Z"
    - type: Connected
      status: "True"
      lastTransitionTime: "2025-01-13T10:30:15Z"
      message: "Connected to 2 relay(s)"
```

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

### From Source

```bash
# Clone repository
git clone https://github.com/YOUR_ORG/portal-expose-controller.git
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
helm repo add portal-expose https://YOUR_ORG.github.io/portal-expose-controller
helm install portal-expose portal-expose/portal-expose
```

## Configuration

### Controller Configuration

The controller can be configured via environment variables or command-line flags:

| Variable | Default | Description |
|----------|---------|-------------|
| `TUNNEL_IMAGE` | `portal-tunnel:latest` | Default tunnel container image |
| `DEFAULT_RELAY_URL` | `wss://portal.gosuda.org/relay` | Default relay if not specified |
| `METRICS_ADDR` | `:8080` | Metrics endpoint address |
| `HEALTH_PROBE_ADDR` | `:8081` | Health probe endpoint address |

### RBAC Permissions

The controller requires the following permissions:

- `portalexposes`: all verbs (create, get, list, watch, update, delete)
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
│       └── portalexpose_types.go    # CRD definitions
├── controllers/
│   └── portalexpose_controller.go   # Main controller logic
├── config/
│   ├── crd/                         # CRD manifests
│   ├── rbac/                        # RBAC configurations
│   └── manager/                     # Controller deployment
├── pkg/
│   └── tunnel/                      # Tunnel management logic
└── main.go                          # Controller entry point
```

## Roadmap

### Phase 1: Core Functionality (Current)
- [x] Project structure and design
- [ ] PortalExpose CRD implementation
- [ ] Basic controller logic
- [ ] Tunnel deployment management
- [ ] Status reporting
- [ ] E2E testing

### Phase 2: Advanced Features
- [ ] Multi-relay failover
- [ ] Automatic relay selection
- [ ] Metrics and monitoring (Prometheus)
- [ ] Helm chart
- [ ] Horizontal tunnel scaling

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
