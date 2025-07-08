# Monitoring Configuration Examples

This directory contains configuration files and examples for monitoring the tektoncd-pruner.

## Files

### grafana-dashboard.json
A ready-to-import Grafana dashboard for visualizing tektoncd-pruner metrics. This dashboard includes:

- Resource processing and deletion rates
- Performance metrics (reconciliation duration)
- Error rate tracking
- Resource age analysis
- Active resource counts

### prometheus-config.yaml
Standard Prometheus configuration for scraping tektoncd-pruner metrics. Use this if you don't have Prometheus Operator installed. Includes:

- Static and Kubernetes service discovery configurations
- Metric filtering and relabeling
- Example alerting rules for common issues

### kind-setup.yaml
Complete Kubernetes manifests for deploying Prometheus and Grafana in a Kind cluster, including:

- Prometheus deployment with tektoncd-pruner configuration
- Grafana deployment with pre-configured dashboards
- RBAC and service accounts
- NodePort services for easy access
- Alerting rules for common issues

### KIND_SETUP.md
Comprehensive guide for setting up monitoring in Kind clusters with step-by-step instructions.

## Usage

### For Kind Clusters (Recommended for Development)

```bash
# Deploy complete monitoring stack
kubectl apply -f examples/monitoring/kind-setup.yaml

# Access Prometheus at http://localhost:30090
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

# Access Grafana at http://localhost:30300 (admin/admin)
kubectl port-forward svc/grafana 3000:3000 -n monitoring
```

See [KIND_SETUP.md](KIND_SETUP.md) for detailed instructions.

### For Production Clusters

#### Import to Grafana via UI:
1. Open Grafana
2. Go to **Dashboards** > **Import**
3. Upload the `grafana-dashboard.json` file
4. Configure the Prometheus data source

#### Import via kubectl (for Grafana running in Kubernetes):
```bash
kubectl create configmap tekton-pruner-dashboard \
  --from-file=examples/monitoring/grafana-dashboard.json \
  -n monitoring

# If using Grafana sidecar for auto-discovery, add labels:
kubectl label configmap tekton-pruner-dashboard \
  grafana_dashboard=1 \
  -n monitoring
```

#### Configure Prometheus (if not using Prometheus Operator):
```bash
# Add the configuration from prometheus-config.yaml to your prometheus.yml
# Or use it as a reference for configuring your Prometheus setup

# For standard Prometheus installation:
# 1. Copy the scrape_configs section to your prometheus.yml
# 2. Restart Prometheus to reload configuration

# For Prometheus in Kubernetes:
kubectl create configmap prometheus-config \
  --from-file=examples/monitoring/prometheus-config.yaml \
  -n monitoring

# Then mount this config in your Prometheus deployment
```

## Prerequisites

- **Prometheus server** configured to scrape tektoncd-pruner metrics
  - Either with Prometheus Operator (use `config/optional/servicemonitor.yaml`)
  - Or standard Prometheus (use `examples/monitoring/prometheus-config.yaml`)
  - Or Kind cluster setup (use `examples/monitoring/kind-setup.yaml`)
- **Grafana** with Prometheus data source configured
- **tektoncd-pruner** running with observability enabled

## Related Configuration

The Kubernetes manifests for observability are located in different directories:
- `config/metrics-service.yaml` - Service to expose metrics (required)
- `config/config-observability.yaml` - Observability configuration (required)
- `config/optional/servicemonitor.yaml` - ServiceMonitor for Prometheus Operator (optional)
- `examples/monitoring/prometheus-config.yaml` - Standard Prometheus configuration (alternative)
- `examples/monitoring/kind-setup.yaml` - Complete Kind cluster setup (development)

## Quick Start Options

| Environment | Method | Files | Description |
|-------------|---------|-------|-------------|
| Kind/Development | All-in-one | `kind-setup.yaml` | Complete monitoring stack |
| Production + Prometheus Operator | ServiceMonitor | `config/optional/servicemonitor.yaml` | Automatic discovery |
| Production + Standard Prometheus | Manual Config | `prometheus-config.yaml` | Manual configuration |
| Existing Grafana | Dashboard Import | `grafana-dashboard.json` | Dashboard only |

## Documentation

For complete setup instructions, see:
- [KIND_SETUP.md](KIND_SETUP.md) - Kind cluster setup guide
- [../../docs/observability.md](../../docs/observability.md) - Complete observability documentation 