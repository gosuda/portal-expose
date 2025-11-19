# Portal Expose Controller - Installation Guide

## Quick Install

Install the Portal Expose Controller to your Kubernetes cluster with a single command:

```bash
kubectl apply -f install.yaml
```

This will create:
- **Namespace**: `portal-expose-system`
- **CRDs**: `PortalExpose` and `TunnelClass`
- **RBAC**: ServiceAccount, ClusterRole, and ClusterRoleBinding
- **Deployment**: Controller running `ghcr.io/gosuda/portal-expose-controller:0.1.0`

## Verify Installation

Check that the controller is running:

```bash
# Check the deployment
kubectl get deployment -n portal-expose-system

# Check the pods
kubectl get pods -n portal-expose-system

# View controller logs
kubectl logs -n portal-expose-system -l app.kubernetes.io/name=portal-expose-controller -f
```

## Create a TunnelClass

Before creating PortalExpose resources, you need at least one TunnelClass:

```bash
kubectl apply -f - <<EOF
apiVersion: portal.gosuda.org/v1alpha1
kind: TunnelClass
metadata:
  name: default
  annotations:
    portal.gosuda.org/is-default-class: "true"
spec:
  replicas: 1
  size: small
EOF
```

## Example Usage

### 1. Deploy a sample application

```bash
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80
```

### 2. Create a PortalExpose resource

```bash
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
```

### 3. Check the status

```bash
# View PortalExpose status
kubectl get portalexpose my-nginx -o yaml

# Check tunnel pods
kubectl get pods -l portal.gosuda.org/portalexpose=my-nginx

# Get the public URL
kubectl get portalexpose my-nginx -o jsonpath='{.status.publicURL}'
```

## Uninstall

To remove the Portal Expose Controller from your cluster:

```bash
# Delete all PortalExpose resources first
kubectl delete portalexpose --all

# Delete all TunnelClasses
kubectl delete tunnelclass --all

# Remove the controller
kubectl delete -f install.yaml
```

## Configuration

### Controller Image

The controller uses:
- **Image**: `ghcr.io/gosuda/portal-expose-controller:0.1.0`
- **Platforms**: `linux/amd64`, `linux/arm64`

### Tunnel Image

Tunnel pods use:
- **Image**: `ghcr.io/gosuda/portal-tunnel:1.0.0`

### Resource Limits

The controller deployment has:
- **Requests**: 10m CPU, 64Mi memory
- **Limits**: 500m CPU, 128Mi memory

## Troubleshooting

### Controller not starting

```bash
# Check pod status
kubectl describe pod -n portal-expose-system -l app.kubernetes.io/name=portal-expose-controller

# View logs
kubectl logs -n portal-expose-system -l app.kubernetes.io/name=portal-expose-controller
```

### PortalExpose stuck in Pending

```bash
# Check if TunnelClass exists
kubectl get tunnelclass

# Check PortalExpose status
kubectl describe portalexpose <name>

# Check tunnel pod status
kubectl get pods -l portal.gosuda.org/portalexpose=<name>
```

### Tunnel pods not starting

```bash
# Check deployment
kubectl get deployment <portalexpose-name>-tunnel

# Check pod events
kubectl describe pod -l portal.gosuda.org/portalexpose=<name>
```

## Support

For issues and questions:
- GitHub: https://github.com/gosuda/portal-expose
- Documentation: See README.md in the repository
