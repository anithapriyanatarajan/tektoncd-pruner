# Optional Configuration Files

This directory contains Kubernetes manifests that require additional components to be installed in your cluster.

## Files

### servicemonitor.yaml
**Requires: Prometheus Operator**

A ServiceMonitor resource for automatic metrics discovery by Prometheus Operator. 

**When to use:**
- You have Prometheus Operator installed in your cluster
- You want automatic metrics scraping configuration
- You're using the Prometheus Operator ecosystem (kube-prometheus-stack, etc.)

**How to apply:**
```bash
# Only apply if you have Prometheus Operator installed
kubectl apply -f config/optional/servicemonitor.yaml
```

**Check if you have Prometheus Operator:**
```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

If this command succeeds, you have Prometheus Operator and can use the ServiceMonitor.

## Alternative Configurations

If you don't have Prometheus Operator, you can still monitor tektoncd-pruner using standard Prometheus configuration.

### Standard Prometheus Configuration

Add this job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'tekton-pruner'
    static_configs:
      - targets: ['tekton-pruner-controller-metrics.tekton-pipelines.svc.cluster.local:9090']
    metrics_path: /metrics
    scrape_interval: 30s
    scrape_timeout: 10s
```

### Kubernetes Service Discovery

For dynamic discovery without Prometheus Operator:

```yaml
scrape_configs:
  - job_name: 'tekton-pruner'
    kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
            - tekton-pipelines
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: tekton-pruner-controller-metrics
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: http-metrics
```

## Installation Instructions

### Installing Prometheus Operator

If you want to use ServiceMonitor, install Prometheus Operator:

```bash
# Using Helm (recommended)
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/kube-prometheus-stack

# Or using manifests
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml
```

After installation, you can apply the ServiceMonitor:
```bash
kubectl apply -f config/optional/servicemonitor.yaml
``` 