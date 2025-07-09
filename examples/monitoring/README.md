# Monitoring Configuration Examples

This directory contains configuration files and examples for monitoring the tektoncd-pruner.

## 🚀 Quick Start with Makefile

### One Command Setup (Recommended for Development)
```bash
# Create Kind cluster + Tekton Pipelines + tektoncd-pruner + monitoring (everything!)
make dev-setup-with-monitoring
```

### Minimal Setup (Without Monitoring)
```bash
# Create Kind cluster + Tekton Pipelines + tektoncd-pruner (no monitoring)
make dev-setup-minimal
```

### Individual Components
```bash
# Deploy Tekton Pipelines first (required)
make deploy-tekton

# Deploy only tektoncd-pruner (requires Tekton)
make apply

# Deploy only monitoring stack
make deploy-monitoring

# Deploy Tekton + tektoncd-pruner together  
make deploy-tekton-with-pruner

# Deploy Tekton + tektoncd-pruner + monitoring together
make deploy-all-with-monitoring

# Check status of all components
make status-all
```

**📋 For all Makefile monitoring commands, see [MAKEFILE_COMMANDS.md](MAKEFILE_COMMANDS.md)**

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

### MAKEFILE_COMMANDS.md
Complete reference guide for all Makefile monitoring automation commands with usage scenarios and troubleshooting.

## Usage

### For Kind Clusters (Recommended for Development)

#### Automated Setup (Easiest)
```bash
# One command - creates everything from scratch
make dev-setup-with-monitoring

# Or minimal setup without monitoring
make dev-setup-minimal

# Or if you already have a cluster
make deploy-all-with-monitoring
```

#### Manual Setup
```bash
# Deploy Tekton Pipelines first
make deploy-tekton

# Deploy tektoncd-pruner
make apply

# Deploy complete monitoring stack
kubectl apply -f examples/monitoring/kind-setup.yaml

# Access Prometheus at http://localhost:30090
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

# Access Grafana at http://localhost:30300 (admin/admin)
kubectl port-forward svc/grafana 3000:3000 -n monitoring
```

See [KIND_SETUP.md](KIND_SETUP.md) for detailed instructions.

### For Production Clusters

#### Prerequisites
- **Tekton Pipelines must be installed first**
- **tektoncd-pruner must be deployed**
- **Prometheus server** configured to scrape tektoncd-pruner metrics
  - Either with Prometheus Operator (use `config/optional/servicemonitor.yaml`)
  - Or standard Prometheus (use `examples/monitoring/prometheus-config.yaml`)
  - Or Kind cluster setup (use `examples/monitoring/kind-setup.yaml`)
- **Grafana** with Prometheus data source configured

#### Quick Production Setup
```bash
# For new environments
make deploy-tekton
make apply

# For environments with existing Tekton
make apply

# For Prometheus Operator environments
make deploy-tekton
make apply-all  # Includes ServiceMonitor
```

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

- **Tekton Pipelines** must be installed first
- **tektoncd-pruner** running with observability enabled
- **Prometheus server** configured to scrape tektoncd-pruner metrics
  - Either with Prometheus Operator (use `config/optional/servicemonitor.yaml`)
  - Or standard Prometheus (use `examples/monitoring/prometheus-config.yaml`)
  - Or Kind cluster setup (use `examples/monitoring/kind-setup.yaml`)
- **Grafana** with Prometheus data source configured

## Related Configuration

The Kubernetes manifests for observability are located in different directories:
- `config/metrics-service.yaml` - Service to expose metrics (required)
- `config/config-observability.yaml` - Observability configuration (required)
- `config/optional/servicemonitor.yaml` - ServiceMonitor for Prometheus Operator (optional)
- `examples/monitoring/prometheus-config.yaml` - Standard Prometheus configuration (alternative)
- `examples/monitoring/kind-setup.yaml` - Complete Kind cluster setup (development)

## Quick Start Options

| Environment | Method | Files/Commands | Description |
|-------------|---------|-------|-------------|
| Kind/Development | Makefile | `make dev-setup-with-monitoring` | Complete automated setup |
| Kind/Development | Makefile | `make dev-setup-minimal` | Minimal setup without monitoring |
| Kind/Development | Manual | `kind-setup.yaml` | Complete monitoring stack |
| Production + Prometheus Operator | Makefile | `make deploy-tekton && make apply-all` | Automatic discovery |
| Production + Standard Prometheus | Manual Config | `prometheus-config.yaml` | Manual configuration |
| Existing Grafana | Dashboard Import | `grafana-dashboard.json` | Dashboard only |

## Monitoring Commands Reference

For comprehensive documentation of all Makefile monitoring commands, see:
- **[MAKEFILE_COMMANDS.md](MAKEFILE_COMMANDS.md)** - Complete command reference with examples

### Most Common Commands
```bash
# Complete setup from scratch
make dev-setup-with-monitoring

# Minimal setup without monitoring
make dev-setup-minimal

# Deploy Tekton + tektoncd-pruner + monitoring to existing cluster
make deploy-all-with-monitoring

# Check status of everything
make status-all

# Debug issues
make logs-tekton
make logs-monitoring

# Clean up
make clean-all
```

## Documentation

For complete setup instructions, see:
- [MAKEFILE_COMMANDS.md](MAKEFILE_COMMANDS.md) - Makefile automation commands
- [KIND_SETUP.md](KIND_SETUP.md) - Kind cluster setup guide
- [../../docs/observability.md](../../docs/observability.md) - Complete observability documentation 