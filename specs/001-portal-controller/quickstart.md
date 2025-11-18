# Quickstart: Portal Expose Kubernetes Controller

**Feature**: Portal Expose Kubernetes Controller
**Date**: 2025-01-18
**Purpose**: Step-by-step guide for developers to validate the implementation

## Prerequisites

- Go 1.21+ installed
- Docker installed (for building controller image)
- kubectl configured with access to a Kubernetes cluster (v1.19+)
- Kubebuilder 3.x+ installed (optional, for code generation)

## Step 1: Set Up Development Environment

### Clone Repository

```bash
git clone https://github.com/gosuda/portal-expose.git
cd portal-expose
git checkout 001-portal-controller
```

### Install Dependencies

```bash
go mod download
```

### Verify Kubebuilder Installation (Optional)

```bash
kubebuilder version
# Should show v3.x or later
```

## Step 2: Generate CRDs and Code

```bash
# Generate CRD YAML manifests from Go types
make manifests

# Generate DeepCopy methods
make generate
```

**Expected Output**:
- `config/crd/bases/portal.gosuda.org_portalexposes.yaml`
- `config/crd/bases/portal.gosuda.org_tunnelclasses.yaml`
- `api/v1alpha1/zz_generated.deepcopy.go`

## Step 3: Run Unit Tests

```bash
make test
```

**Expected Output**:
- All tests pass
- Coverage report shows >80% coverage for reconciliation logic

**Key Test Files**:
- `controllers/portalexpose_controller_test.go`
- `controllers/tunnelclass_controller_test.go`
- `internal/tunnel/deployment_test.go`
- `internal/tunnel/status_test.go`

## Step 4: Run Integration Tests (envtest)

```bash
# Integration tests run against a local test API server (envtest)
make test-integration
```

**Test Scenarios Validated**:
1. PortalExpose creation triggers tunnel Deployment creation
2. TunnelClass update triggers Deployment rolling update
3. PortalExpose deletion cleans up tunnel Deployment
4. Status updates reflect pod readiness and relay states
5. Degraded phase on partial pod/relay failures
6. Failed phase when Service not found

## Step 5: Install CRDs to Cluster

```bash
# Install PortalExpose and TunnelClass CRDs
make install
```

**Verify CRDs Installed**:
```bash
kubectl get crd | grep portal.gosuda.org
# Should show:
# portalexposes.portal.gosuda.org
# tunnelclasses.portal.gosuda.org
```

## Step 6: Run Controller Locally

```bash
# Run controller on local machine (watches cluster API server)
make run
```

**Expected Output**:
```
INFO    controller-runtime.metrics    Starting metrics server
INFO    controller-runtime.manager    Starting manager
INFO    Starting EventSource    controller=portalexpose
INFO    Starting Controller     controller=portalexpose
INFO    Starting workers        controller=portalexpose worker count=1
```

**Note**: Controller runs locally but manages resources in the cluster.

## Step 7: Create Test Resources

### Create a Test Service

```bash
# Create a simple HTTP service to expose
kubectl create deployment hello-app --image=gcr.io/google-samples/hello-app:1.0
kubectl expose deployment hello-app --port=8080
```

### Create Default TunnelClass

```bash
cat <<EOF | kubectl apply -f -
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

### Create PortalExpose

```bash
cat <<EOF | kubectl apply -f -
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
      - name: gosuda-portal
        url: wss://portal.gosuda.org/relay
EOF
```

## Step 8: Verify Controller Behavior

### Check PortalExpose Status

```bash
kubectl get portalexpose hello-app -o yaml
```

**Expected Status**:
```yaml
status:
  phase: Pending  # or Ready if tunnel pods started successfully
  publicURL: https://hello-app.portal.gosuda.org
  tunnelPods:
    ready: 0  # or 1 when pod is ready
    total: 1
  conditions:
    - type: ServiceExists
      status: "True"
      reason: ServiceFound
    - type: TunnelDeploymentReady
      status: "False"  # or True when pods ready
      reason: PodsNotReady
```

### Check Tunnel Deployment Created

```bash
kubectl get deployment hello-app-tunnel
```

**Expected Output**:
```
NAME                 READY   UP-TO-DATE   AVAILABLE   AGE
hello-app-tunnel     0/1     1            0           10s
```

### Check Tunnel Pods

```bash
kubectl get pods -l portal.gosuda.org/portalexpose=hello-app
```

**Expected Output**:
```
NAME                                READY   STATUS              RESTARTS   AGE
hello-app-tunnel-xxxxxxxxxx-xxxxx   0/1     ContainerCreating   0          15s
```

**Note**: Pods may fail if tunnel image is not yet available. For quickstart, you can simulate by checking controller logs and status updates.

### Watch Controller Logs

```bash
# In terminal running "make run", observe logs:
INFO    Reconciling PortalExpose    name=hello-app namespace=default
INFO    Creating tunnel Deployment  name=hello-app-tunnel
INFO    Updated PortalExpose status phase=Pending
```

## Step 9: Test TunnelClass Update (Rolling Update)

### Update TunnelClass

```bash
kubectl patch tunnelclass default --type=merge -p '{"spec":{"size":"medium"}}'
```

### Verify Rolling Update

```bash
# Watch Deployment for rolling update
kubectl rollout status deployment hello-app-tunnel

# Check PortalExpose status shows Progressing
kubectl get portalexpose hello-app -o jsonpath='{.status.conditions[?(@.type=="Progressing")]}'
```

**Expected**: Deployment performs rolling update, replacing pods with new resource requests (medium size).

## Step 10: Test Deletion Cleanup

### Delete PortalExpose

```bash
kubectl delete portalexpose hello-app
```

### Verify Cleanup

```bash
# Tunnel Deployment should be deleted
kubectl get deployment hello-app-tunnel
# Expected: Error from server (NotFound)

# PortalExpose should be fully deleted (no finalizer blocking)
kubectl get portalexpose hello-app
# Expected: Error from server (NotFound)
```

## Step 11: Build and Deploy Controller to Cluster

### Build Controller Image

```bash
# Build Docker image
make docker-build IMG=your-registry/portal-expose-controller:v0.1.0

# Push to registry
make docker-push IMG=your-registry/portal-expose-controller:v0.1.0
```

### Deploy Controller

```bash
# Deploy controller as Deployment in cluster
make deploy IMG=your-registry/portal-expose-controller:v0.1.0
```

**Verify Deployment**:
```bash
kubectl get deployment -n portal-expose-system portal-expose-controller-manager
kubectl logs -n portal-expose-system deployment/portal-expose-controller-manager -f
```

### Repeat Tests with Deployed Controller

Re-run Step 7-10 tests with controller running in-cluster instead of locally.

## Step 12: Run End-to-End Tests

### Prerequisites

- Real Portal relay accessible (e.g., `wss://portal.gosuda.org/relay`)
- Portal tunnel image available

### Run E2E Suite

```bash
make test-e2e
```

**E2E Test Scenarios**:
1. Create PortalExpose → verify tunnel pods connect to relay
2. Access public URL → verify traffic reaches Kubernetes Service
3. Multi-relay setup → verify connections to all relays
4. Simulate relay failure → verify Degraded phase
5. Delete PortalExpose → verify cleanup

## Step 13: Validate Success Criteria

Review specification success criteria (spec.md) and verify:

- ✅ **SC-001**: PortalExpose creation to tunnel pod deployment <30 seconds
- ✅ **SC-002**: PortalExpose deletion cleanup <10 seconds
- ✅ **SC-003**: Status updates within 5 seconds of state changes
- ✅ **SC-004**: Controller handles 100+ PortalExposes (load test)
- ✅ **SC-005**: 95% creation success rate (when Service exists and relays reachable)
- ✅ **SC-006**: Multi-relay availability with 50% relay failures
- ✅ **SC-007**: TunnelClass changes propagate within 60 seconds
- ✅ **SC-008**: Operators diagnose issues via status fields and Events
- ✅ **SC-009**: Controller crash recovery without orphaned resources
- ✅ **SC-010**: Error messages are actionable

## Troubleshooting

### CRD Not Found Errors

```bash
# Ensure CRDs are installed
make install

# Verify CRDs exist
kubectl get crd portalexposes.portal.gosuda.org
```

### Controller Not Starting

```bash
# Check RBAC permissions
kubectl get clusterrole portal-expose-controller-manager-role

# Check controller logs
kubectl logs -n portal-expose-system deployment/portal-expose-controller-manager
```

### Tunnel Pods Not Starting

```bash
# Check Deployment spec
kubectl get deployment hello-app-tunnel -o yaml

# Check pod events
kubectl describe pod -l portal.gosuda.org/portalexpose=hello-app

# Verify tunnel image is accessible
docker pull ghcr.io/gosuda/portal-tunnel:latest
```

### Status Not Updating

```bash
# Check controller logs for reconciliation errors
kubectl logs -n portal-expose-system deployment/portal-expose-controller-manager | grep "Reconciling PortalExpose"

# Verify controller is running
kubectl get pods -n portal-expose-system
```

### Service Not Found Errors

```bash
# Verify Service exists in same namespace as PortalExpose
kubectl get service hello-app

# Check PortalExpose conditions
kubectl get portalexpose hello-app -o jsonpath='{.status.conditions[?(@.type=="ServiceExists")]}'
```

## Next Steps

After validating quickstart:

1. Run `/speckit.tasks` to generate detailed implementation tasks
2. Implement controllers following task breakdown
3. Write tests for each task (TDD workflow)
4. Submit PR for code review
5. Deploy to staging cluster for integration testing

## Reference Commands

### Useful kubectl Commands

```bash
# Watch PortalExposes
kubectl get portalexpose -w

# Describe PortalExpose (shows Events)
kubectl describe portalexpose hello-app

# Get status in JSON format
kubectl get portalexpose hello-app -o json | jq '.status'

# List all tunnel Deployments
kubectl get deployments -l app.kubernetes.io/managed-by=portal-expose-controller

# Check controller manager metrics
kubectl port-forward -n portal-expose-system svc/portal-expose-controller-manager-metrics-service 8080:8443
curl http://localhost:8080/metrics
```

### Makefile Targets

- `make manifests`: Generate CRD YAML
- `make generate`: Generate DeepCopy code
- `make test`: Run unit tests
- `make test-integration`: Run integration tests with envtest
- `make test-e2e`: Run end-to-end tests
- `make install`: Install CRDs to cluster
- `make uninstall`: Remove CRDs from cluster
- `make deploy`: Deploy controller to cluster
- `make undeploy`: Remove controller from cluster
- `make run`: Run controller locally
- `make docker-build`: Build controller Docker image
- `make docker-push`: Push image to registry
