# Setting up Prometheus Monitoring in Kind Cluster

This guide walks you through setting up Prometheus monitoring for tektoncd-pruner in a Kind cluster.

## Prerequisites

- Kind cluster running
- kubectl configured to use the Kind cluster
- tektoncd-pruner deployed and running

## 🚀 Quick Setup

### Step 1: Deploy Monitoring Stack

```bash
# Apply the complete monitoring setup
kubectl apply -f examples/monitoring/kind-setup.yaml

# Wait for pods to be ready
kubectl wait --for=condition=Ready pod -l app=prometheus -n monitoring --timeout=300s
kubectl wait --for=condition=Ready pod -l app=grafana -n monitoring --timeout=300s
```

### Step 2: Verify Installation

```bash
# Check pods are running
kubectl get pods -n monitoring

# Check services
kubectl get svc -n monitoring

# Check that tektoncd-pruner metrics service is accessible
kubectl get svc tekton-pruner-controller-metrics -n tekton-pipelines
```

### Step 3: Access Prometheus

```bash
# Get the Kind cluster IP
CLUSTER_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')

# Access Prometheus Web UI
echo "Prometheus available at: http://$CLUSTER_IP:30090"

# Or use port-forward
kubectl port-forward svc/prometheus 9090:9090 -n monitoring
# Then visit http://localhost:9090
```

### Step 4: Access Grafana

```bash
# Access Grafana Web UI
echo "Grafana available at: http://$CLUSTER_IP:30300"
# Default credentials: admin/admin

# Or use port-forward
kubectl port-forward svc/grafana 3000:3000 -n monitoring
# Then visit http://localhost:3000
```

## 📊 Configuring Grafana

### 1. Add Prometheus Data Source

1. Open Grafana at http://localhost:3000
2. Login with admin/admin
3. Go to **Configuration** → **Data Sources**
4. Click **Add data source**
5. Select **Prometheus**
6. Set URL to: `http://prometheus:9090`
7. Click **Save & Test**

### 2. Import Tektoncd-Pruner Dashboard

1. Go to **Dashboards** → **Import**
2. Upload the file: `examples/monitoring/grafana-dashboard.json`
3. Select the Prometheus data source
4. Click **Import**

## 🔍 Verification

### Check Prometheus Targets

1. Open Prometheus Web UI
2. Go to **Status** → **Targets**
3. Verify `tekton-pruner` target is **UP**

### Sample Prometheus Queries

```promql
# Resource processing rate
rate(tektoncd_pruner_resources_processed_total[5m])

# Error rate
rate(tektoncd_pruner_resources_errors_total[5m])

# Reconciliation duration
histogram_quantile(0.95, rate(tektoncd_pruner_reconciliation_duration_seconds_bucket[5m]))

# Active resources
tektoncd_pruner_active_resources
```

## 🛠️ Troubleshooting

### Prometheus Can't Scrape Tektoncd-Pruner

1. **Check service exists:**
   ```bash
   kubectl get svc tekton-pruner-controller-metrics -n tekton-pipelines
   ```

2. **Test metrics endpoint:**
   ```bash
   kubectl port-forward svc/tekton-pruner-controller-metrics 9090:9090 -n tekton-pipelines
   curl http://localhost:9090/metrics | grep tektoncd_pruner
   ```

3. **Check Prometheus logs:**
   ```bash
   kubectl logs -l app=prometheus -n monitoring
   ```

### Grafana Can't Connect to Prometheus

1. **Check Prometheus service:**
   ```bash
   kubectl get svc prometheus -n monitoring
   ```

2. **Test connection from Grafana pod:**
   ```bash
   kubectl exec -it deployment/grafana -n monitoring -- nc -zv prometheus 9090
   ```

### No Metrics Showing

1. **Verify tektoncd-pruner configuration:**
   ```bash
   kubectl get configmap config-observability-tekton-pruner -n tekton-pipelines -o yaml
   ```

2. **Check controller logs:**
   ```bash
   kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines
   ```

3. **Verify environment variables:**
   ```bash
   kubectl get deployment tekton-pruner-controller -n tekton-pipelines -o yaml | grep -A 10 -B 10 METRICS_ENABLED
   ```

## 🎯 Alternative Setup Methods

### Using Helm (Alternative)

If you prefer using Helm:

```bash
# Add Prometheus community repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Install Prometheus and Grafana
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.additionalScrapeConfigs[0].job_name=tekton-pruner \
  --set prometheus.prometheusSpec.additionalScrapeConfigs[0].static_configs[0].targets[0]="tekton-pruner-controller-metrics.tekton-pipelines.svc.cluster.local:9090"
```

### Manual ConfigMap Update

To update just the Prometheus configuration:

```bash
# Edit the prometheus config
kubectl edit configmap prometheus-config -n monitoring

# Reload Prometheus configuration
kubectl exec -it deployment/prometheus -n monitoring -- kill -HUP 1
```

## 📋 Clean Up

```bash
# Remove monitoring stack
kubectl delete -f examples/monitoring/kind-setup.yaml

# Or delete namespace
kubectl delete namespace monitoring
```

## 🔗 Related Resources

- [Prometheus Configuration](prometheus-config.yaml)
- [Grafana Dashboard](grafana-dashboard.json)
- [Observability Documentation](../../docs/observability.md)
- [Optional ServiceMonitor](../config/optional/servicemonitor.yaml) 