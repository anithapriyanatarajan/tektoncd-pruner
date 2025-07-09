# Makefile Monitoring Commands Reference

This document provides a comprehensive reference for all Makefile commands related to monitoring deployment and management.

## 🚀 Quick Start Commands

### Complete Setup (One Command)
```bash
# Setup Kind cluster + Tekton Pipelines + tektoncd-pruner + monitoring (everything!)
make dev-setup-with-monitoring
```

This command will:
1. Create Kind cluster with local registry
2. Deploy Tekton Pipelines
3. Deploy tektoncd-pruner core components
4. Deploy complete monitoring stack (Prometheus + Grafana)
5. Wait for all pods to be ready
6. Show access instructions

### Minimal Setup (Without Monitoring)
```bash
# Setup Kind cluster + Tekton Pipelines + tektoncd-pruner (no monitoring)
make dev-setup-minimal
```

This command will:
1. Create Kind cluster with local registry
2. Deploy Tekton Pipelines
3. Deploy tektoncd-pruner core components
4. Wait for all pods to be ready
5. Show access instructions

### Individual Deployment
```bash
# Deploy Tekton Pipelines first (required)
make deploy-tekton

# Deploy only tektoncd-pruner core (requires Tekton)
make apply

# Deploy complete monitoring stack
make deploy-monitoring

# Deploy Tekton + tektoncd-pruner together (no monitoring)
make deploy-tekton-with-pruner

# Deploy Tekton + tektoncd-pruner + monitoring together
make deploy-all-with-monitoring
```

## 📋 All Available Commands

| Command | Description | Use Case |
|---------|-------------|----------|
| `make deploy-tekton` | Deploy Tekton Pipelines | Prerequisites for tektoncd-pruner |
| `make apply` | Deploy core tektoncd-pruner (no optional) | Most environments (requires Tekton) |
| `make apply-all` | Deploy tektoncd-pruner + ServiceMonitor | With Prometheus Operator (requires Tekton) |
| `make apply-optional` | Deploy only ServiceMonitor | Add to existing deployment |
| `make deploy-monitoring` | Deploy Prometheus + Grafana | Development/testing |
| `make deploy-tekton-with-pruner` | Deploy Tekton + tektoncd-pruner | Complete setup without monitoring |
| `make deploy-all-with-monitoring` | Deploy Tekton + tektoncd-pruner + monitoring | Complete setup with monitoring |
| `make dev-setup-with-monitoring` | Kind + Tekton + tektoncd-pruner + monitoring | From scratch development |
| `make dev-setup-minimal` | Kind + Tekton + tektoncd-pruner | From scratch without monitoring |
| `make status-tekton` | Check Tekton Pipelines status | Troubleshooting |
| `make status-monitoring` | Check monitoring stack status | Troubleshooting |
| `make status-all` | Check all components status | Complete system check |
| `make logs-tekton` | Show Tekton Pipelines logs | Debugging |
| `make logs-monitoring` | Show monitoring logs | Debugging |
| `make clean-tekton` | Remove Tekton Pipelines | Cleanup |
| `make clean-monitoring` | Remove monitoring stack | Cleanup |
| `make clean-all` | Remove everything | Full cleanup |

## 🎯 Usage Scenarios

### Scenario 1: Development Environment (From Scratch)
```bash
# One command setup with monitoring
make dev-setup-with-monitoring

# Or minimal setup without monitoring
make dev-setup-minimal

# Access services
kubectl port-forward svc/prometheus 9090:9090 -n monitoring
kubectl port-forward svc/grafana 3000:3000 -n monitoring
```

### Scenario 2: Existing Cluster - Add Complete Stack
```bash
# Deploy Tekton Pipelines first
make deploy-tekton

# Deploy tektoncd-pruner
make apply

# Add monitoring stack
make deploy-monitoring

# Check everything is working
make status-all
```

### Scenario 3: Existing Cluster - One Command
```bash
# Deploy everything together
make deploy-all-with-monitoring

# Check status
make status-all
```

### Scenario 4: Production with Prometheus Operator
```bash
# Deploy Tekton first
make deploy-tekton

# Deploy everything including ServiceMonitor
make apply-all

# ServiceMonitor will be automatically discovered by Prometheus Operator
```

### Scenario 5: Testing/Debugging
```bash
# Check status of everything
make status-all

# View logs if there are issues
make logs-tekton
make logs-monitoring

# Clean up and restart
make clean-all
make dev-setup-with-monitoring
```

## 🔍 Command Details

### `make deploy-tekton`
**What it does:**
- Deploys Tekton Pipelines from the latest release
- Waits for Tekton controller and webhook pods to be ready
- Shows success message

**Output:**
```
🐱 deploying Tekton Pipelines
🐱 Waiting for Tekton Pipelines to be ready...
🐱 Tekton Pipelines deployed successfully!
```

### `make deploy-tekton-with-pruner`
**What it does:**
- Runs `make deploy-tekton` (Tekton Pipelines)
- Runs `make apply` (tektoncd-pruner core)
- Shows verification commands

**Perfect for:** Quick setup without monitoring

### `make dev-setup-minimal`
**What it does:**
- Runs `make dev-setup` (creates Kind cluster)
- Runs `make deploy-tekton-with-pruner`
- Complete minimal environment from scratch

**Perfect for:** Starting development without monitoring overhead

### `make deploy-monitoring`
**What it does:**
- Applies `examples/monitoring/kind-setup.yaml`
- Waits for Prometheus and Grafana pods to be ready
- Shows access instructions

**Output:**
```
🐱 deploying monitoring stack (Prometheus + Grafana)
🐱 Waiting for monitoring pods to be ready...
🐱 Monitoring stack deployed successfully!
🐱 Prometheus: kubectl port-forward svc/prometheus 9090:9090 -n monitoring
🐱 Grafana: kubectl port-forward svc/grafana 3000:3000 -n monitoring (admin/admin)
```

### `make deploy-all-with-monitoring`
**What it does:**
- Runs `make deploy-tekton` (Tekton Pipelines)
- Runs `make apply` (tektoncd-pruner core)
- Runs `make deploy-monitoring` (Prometheus + Grafana)
- Shows verification commands

**Perfect for:** Complete development setup with monitoring

### `make dev-setup-with-monitoring`
**What it does:**
- Runs `make dev-setup` (creates Kind cluster)
- Runs `make deploy-all-with-monitoring`
- Complete environment from scratch with monitoring

**Perfect for:** Starting development from nothing with full observability

### `make status-tekton`
**What it does:**
- Checks Tekton Pipelines namespace
- Shows Tekton Pipelines pods status
- Shows Tekton Pipelines services

**Sample output:**
```
🐱 Tekton Pipelines namespace status:
NAME              STATUS   AGE
tekton-pipelines  Active   5m

🐱 Tekton Pipelines pods:
NAME                                   READY   STATUS    RESTARTS   AGE
tekton-pipelines-controller-xyz123     1/1     Running   0          5m
tekton-pipelines-webhook-abc456        1/1     Running   0          5m

🐱 Tekton Pipelines services:
NAME                          TYPE        CLUSTER-IP     PORT(S)    AGE
tekton-pipelines-controller   ClusterIP   10.96.123.45   9090/TCP   5m
tekton-pipelines-webhook      ClusterIP   10.96.123.46   9443/TCP   5m
```

### `make status-monitoring`
**What it does:**
- Checks monitoring namespace
- Shows monitoring pods status
- Shows monitoring services
- Verifies tektoncd-pruner metrics service

**Sample output:**
```
🐱 Monitoring namespace status:
NAME         STATUS   AGE
monitoring   Active   5m

🐱 Monitoring pods:
NAME                          READY   STATUS    RESTARTS   AGE
grafana-5f8b8b8b8b-xyz12     1/1     Running   0          5m
prometheus-6d7b8b8b8b-abc34  1/1     Running   0          5m

🐱 tektoncd-pruner metrics service:
NAME                              TYPE        CLUSTER-IP     PORT(S)    AGE
tekton-pruner-controller-metrics  ClusterIP   10.96.123.45   9090/TCP   5m
```

### `make status-all`
**What it does:**
- Runs `make status-tekton` (checks Tekton Pipelines)
- Runs `make status-monitoring` (checks monitoring stack)
- Shows overall system status

**Perfect for:** Comprehensive system health check

### `make logs-tekton`
**What it does:**
- Shows last 20 lines of Tekton Pipelines Controller logs
- Shows last 20 lines of Tekton Pipelines Webhook logs
- Useful for troubleshooting Tekton issues

### `make logs-monitoring`
**What it does:**
- Shows last 20 lines of Prometheus logs
- Shows last 20 lines of Grafana logs
- Useful for troubleshooting monitoring issues

### `make clean-tekton`
**What it does:**
- Removes Tekton Pipelines completely
- Uses `--ignore-not-found=true` for safe cleanup
- Does NOT remove tektoncd-pruner

### `make clean-monitoring`
**What it does:**
- Removes monitoring stack completely
- Uses `--ignore-not-found=true` for safe cleanup
- Does NOT remove tektoncd-pruner or Tekton

### `make clean-all`
**What it does:**
- Runs `make clean` (removes build artifacts)
- Runs `make clean-tekton` (removes Tekton Pipelines)
- Runs `make clean-monitoring` (removes monitoring)
- Complete cleanup

## 🛠️ Troubleshooting Commands

### Check if everything is working
```bash
# Check status of all components
make status-all

# Check individual components
make status-tekton
make status-monitoring

# Check logs for errors
make logs-tekton
make logs-monitoring

# Test tektoncd-pruner metrics endpoint
kubectl port-forward svc/tekton-pruner-controller-metrics 9090:9090 -n tekton-pipelines
curl http://localhost:9090/metrics | grep tektoncd_pruner
```

### If deployment fails
```bash
# Clean up and retry
make clean-all
make dev-setup-with-monitoring

# Or check specific component logs
make logs-tekton
make logs-monitoring
kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines
```

### If Tekton Pipelines won't start
```bash
# Check Tekton Pipelines status
make status-tekton

# Check Tekton Pipelines logs
make logs-tekton

# Check resource availability
kubectl top nodes
kubectl describe pods -n tekton-pipelines

# Check events
kubectl get events -n tekton-pipelines --sort-by='.lastTimestamp'
```

### If monitoring pods won't start
```bash
# Check resource availability
kubectl top nodes
kubectl describe pods -n monitoring

# Check events
kubectl get events -n monitoring --sort-by='.lastTimestamp'

# Check monitoring logs
make logs-monitoring
```

### If tektoncd-pruner won't start
```bash
# Check if Tekton is running first
make status-tekton

# Check tektoncd-pruner logs
kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines

# Check tektoncd-pruner configuration
kubectl get configmap -n tekton-pipelines
kubectl describe configmap config-observability-tekton-pruner -n tekton-pipelines
```

## 📊 Access Services

### Prometheus
```bash
# Port-forward
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

# Or NodePort (Kind clusters)
CLUSTER_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
echo "Prometheus: http://$CLUSTER_IP:30090"
```

### Grafana
```bash
# Port-forward
kubectl port-forward svc/grafana 3000:3000 -n monitoring

# Or NodePort (Kind clusters) 
echo "Grafana: http://$CLUSTER_IP:30300"
# Default credentials: admin/admin
```

### tektoncd-pruner Metrics
```bash
# Direct access to metrics
kubectl port-forward svc/tekton-pruner-controller-metrics 9090:9090 -n tekton-pipelines
curl http://localhost:9090/metrics
```

### Tekton Pipelines
```bash
# Check Tekton Pipelines status
kubectl get pods -n tekton-pipelines

# Check Tekton Pipelines logs
kubectl logs deployment/tekton-pipelines-controller -n tekton-pipelines
kubectl logs deployment/tekton-pipelines-webhook -n tekton-pipelines
```

## 🔄 Typical Workflows

### Daily Development
```bash
# Start work
make dev-setup-with-monitoring

# Make changes to code...

# Restart tektoncd-pruner only
make apply

# Full restart if needed
make clean-all
make dev-setup-with-monitoring
```

### Testing Changes
```bash
# Deploy your changes
make apply

# Check all components
make status-all

# Debug if needed
make logs-tekton
make logs-monitoring
```

### Production Deployment
```bash
# For environments with existing Tekton
make apply

# For new environments
make deploy-tekton
make apply

# For Prometheus Operator environments
make deploy-tekton
make apply-all

# For standard Prometheus
make deploy-tekton
make apply
# Then configure Prometheus manually with examples/monitoring/prometheus-config.yaml
```

### Continuous Integration
```bash
# Setup test environment
make dev-setup-minimal

# Run your tests...

# Cleanup
make clean-all
```

## 📝 Notes

- **Tekton Dependency**: tektoncd-pruner requires Tekton Pipelines to be installed first
- **Kind Clusters**: All monitoring commands work best with Kind clusters
- **Resource Requirements**: 
  - Tekton Pipelines: ~200MB RAM, 0.1 CPU cores
  - tektoncd-pruner: ~100MB RAM, 0.1 CPU cores
  - Monitoring stack: ~1GB RAM, 1 CPU core
- **Persistence**: Monitoring data is stored in `emptyDir` volumes (non-persistent)
- **Networking**: NodePort services use ports 30090 (Prometheus) and 30300 (Grafana)
- **Security**: Default Grafana credentials are admin/admin (change in production)

## 🔗 Related Files

- `examples/monitoring/kind-setup.yaml` - Complete monitoring stack
- `examples/monitoring/KIND_SETUP.md` - Detailed setup guide
- `examples/monitoring/prometheus-config.yaml` - Prometheus configuration
- `examples/monitoring/grafana-dashboard.json` - Grafana dashboard
- `config/optional/servicemonitor.yaml` - ServiceMonitor for Prometheus Operator
- `docs/observability.md` - Complete observability documentation 