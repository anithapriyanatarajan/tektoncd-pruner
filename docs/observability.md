# Observability Configuration

This document describes the comprehensive observability features of tektoncd-pruner, including OpenTelemetry metrics, distributed tracing, and monitoring setup.

## Overview

tektoncd-pruner includes built-in observability capabilities powered by OpenTelemetry (OTel), providing:

- **Metrics**: 15+ metric instruments for resource processing, performance monitoring, and system health
- **Tracing**: Distributed tracing with span creation and error tracking
- **Dual Export**: Support for both Prometheus and OTLP endpoints
- **Configuration**: Environment-based configuration for various deployment scenarios

## Quick Start

### 1. Deploy tektoncd-pruner with Observability

```bash
# Deploy core manifests (recommended for most environments)
make apply

# Deploy with optional manifests (requires Prometheus Operator)
make apply-all

# Deploy only optional manifests (if you already have core components)
make apply-optional
```

### 2. Access Metrics

```bash
# Port-forward to metrics endpoint
kubectl port-forward svc/tekton-pruner-controller-metrics 9090:9090 -n tekton-pipelines

# View metrics
curl http://localhost:9090/metrics | grep tektoncd_pruner
```

### 3. Configure Monitoring

Choose one of these monitoring approaches:

- **Kind/Development**: Use `examples/monitoring/kind-setup.yaml` for a complete stack
- **Prometheus Operator**: Use `config/optional/servicemonitor.yaml` for automatic discovery
- **Standard Prometheus**: Use `examples/monitoring/prometheus-config.yaml` for manual configuration

## Deployment Options

### Core Deployment (Recommended)

```bash
# Deploys essential components without optional dependencies
make apply
```

**Includes:**
- Controller deployment with observability enabled
- Metrics service endpoint
- Configuration for metrics and tracing
- RBAC and service accounts

**Does NOT include:**
- ServiceMonitor (requires Prometheus Operator)

### Full Deployment

```bash
# Deploys all components including optional ones
make apply-all
```

**Includes everything from core deployment plus:**
- ServiceMonitor for Prometheus Operator

### Optional Components Only

```bash
# Deploys only optional manifests
make apply-optional
```

**Use this when:**
- You already have core components deployed
- You want to add Prometheus Operator support
- You need to selectively apply optional configurations

## Metrics Reference

### Resource Processing Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_resources_processed_total` | Counter | Total resources processed | `namespace`, `resource_type`, `status` |
| `tektoncd_pruner_resources_deleted_total` | Counter | Total resources deleted | `namespace`, `resource_type`, `reason` |
| `tektoncd_pruner_resources_errors_total` | Counter | Total processing errors | `namespace`, `resource_type`, `reason` |
| `tektoncd_pruner_resources_skipped_total` | Counter | Total resources skipped | `namespace`, `resource_type`, `reason` |

### Performance Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_reconciliation_duration_seconds` | Histogram | Time spent in reconciliation | `namespace`, `resource_type` |
| `tektoncd_pruner_ttl_processing_duration_seconds` | Histogram | Time spent processing TTL | `namespace`, `resource_type` |
| `tektoncd_pruner_history_processing_duration_seconds` | Histogram | Time spent processing history limits | `namespace`, `resource_type` |
| `tektoncd_pruner_resource_deletion_duration_seconds` | Histogram | Time spent deleting resources | `namespace`, `resource_type` |

### State Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_resources_queued_total` | Counter | Total resources queued | `namespace`, `resource_type` |
| `tektoncd_pruner_current_resources_queued` | Gauge | Current resources in queue | `namespace`, `resource_type` |
| `tektoncd_pruner_active_resources_count` | Gauge | Current active resources | `namespace`, `resource_type` |

### TTL-specific Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_ttl_annotation_updates_total` | Counter | TTL annotation updates | `namespace`, `resource_type` |
| `tektoncd_pruner_ttl_expiration_events_total` | Counter | TTL expiration events | `namespace`, `resource_type` |

### History Limit Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_history_limit_events_total` | Counter | History limit events | `namespace`, `resource_type` |
| `tektoncd_pruner_resources_cleaned_by_history` | Counter | Resources cleaned by history limits | `namespace`, `resource_type` |

### Configuration Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_configuration_reloads_total` | Counter | Configuration reloads | `config_level` |
| `tektoncd_pruner_configuration_errors_total` | Counter | Configuration errors | `config_level` |

### Resource Age Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_resource_age_at_deletion_seconds` | Histogram | Age of resources when deleted | `namespace`, `resource_type` |

### Error Breakdown Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_resource_delete_errors_total` | Counter | Resource deletion errors | `namespace`, `resource_type` |
| `tektoncd_pruner_resource_update_errors_total` | Counter | Resource update errors | `namespace`, `resource_type` |

## Configuration

### Environment Variables

The observability features are configured through environment variables in the controller deployment:

```yaml
env:
  # Basic Configuration
  - name: OTEL_SERVICE_NAME
    value: "tektoncd-pruner"
  - name: OTEL_SERVICE_VERSION
    value: "v1.0.0"
  
  # Metrics Configuration
  - name: METRICS_ENABLED
    value: "true"
  - name: METRICS_PORT
    value: "9090"
  - name: PROMETHEUS_ENABLED
    value: "true"
  
  # Tracing Configuration
  - name: TRACING_ENABLED
    value: "true"
  - name: TRACING_SAMPLE_RATE
    value: "0.1"
  
  # OTLP Configuration
  - name: OTLP_METRICS_ENABLED
    value: "false"
  - name: OTLP_TRACE_ENABLED
    value: "false"
  - name: OTLP_ENDPOINT
    value: "http://otel-collector:4317"
  
  # Resource Attributes
  - name: OTEL_RESOURCE_ATTRIBUTES
    value: "service.name=tektoncd-pruner,service.version=v1.0.0"
```

### ConfigMap Settings

Additional configuration is available through the `config-observability` ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-observability-tekton-pruner
  namespace: tekton-pipelines
data:
  # OpenTelemetry Metrics Configuration
  otel.metrics.enabled: "true"
  otel.metrics.port: "9090"
  otel.metrics.interval: "30s"
  otel.metrics.prometheus.enabled: "true"
  otel.metrics.otlp.enabled: "false"
  otel.metrics.otlp.endpoint: "http://otel-collector:4317"
  otel.metrics.otlp.headers: ""
  otel.metrics.otlp.insecure: "false"
  
  # OpenTelemetry Tracing Configuration
  otel.tracing.enabled: "true"
  otel.tracing.sample-rate: "0.1"
  otel.tracing.otlp.enabled: "false"
  otel.tracing.otlp.endpoint: "http://otel-collector:4317"
  otel.tracing.otlp.headers: ""
  otel.tracing.otlp.insecure: "false"
  
  # Custom histogram buckets for duration metrics
  otel.metrics.histograms.reconciliation-duration.buckets: "0.1,0.5,1.0,2.5,5.0,10.0,30.0,60.0,120.0"
  otel.metrics.histograms.processing-duration.buckets: "0.01,0.05,0.1,0.5,1.0,2.5,5.0,10.0"
  otel.metrics.histograms.resource-age.buckets: "300,900,1800,3600,7200,14400,28800,86400,172800,604800"
```

## Monitoring Setup

### Option 1: Kind Cluster (Development)

For development environments, use the complete monitoring stack:

```bash
# Deploy monitoring stack
kubectl apply -f examples/monitoring/kind-setup.yaml

# Access Prometheus
kubectl port-forward svc/prometheus 9090:9090 -n monitoring

# Access Grafana
kubectl port-forward svc/grafana 3000:3000 -n monitoring
```

### Option 2: Prometheus Operator (Production)

For production environments with Prometheus Operator:

```bash
# Deploy core components
make apply

# Deploy ServiceMonitor for automatic discovery
make apply-optional

# Verify ServiceMonitor is picked up
kubectl get servicemonitor -n tekton-pipelines
```

### Option 3: Standard Prometheus (Production)

For production environments with standard Prometheus:

```bash
# Deploy core components
make apply

# Configure Prometheus using examples/monitoring/prometheus-config.yaml
# Add the scrape configuration to your prometheus.yml
```

## Sample Queries

### Resource Processing Rate
```promql
rate(tektoncd_pruner_resources_processed_total[5m])
```

### Error Rate
```promql
rate(tektoncd_pruner_resources_errors_total[5m])
```

### 95th Percentile Reconciliation Duration
```promql
histogram_quantile(0.95, rate(tektoncd_pruner_reconciliation_duration_seconds_bucket[5m]))
```

### Active Resources by Namespace
```promql
tektoncd_pruner_active_resources_count by (namespace)
```

### Resources Deleted by Reason
```promql
rate(tektoncd_pruner_resources_deleted_total[5m]) by (reason)
```

### TTL Processing Performance
```promql
histogram_quantile(0.99, rate(tektoncd_pruner_ttl_processing_duration_seconds_bucket[5m]))
```

### Configuration Reload Events
```promql
rate(tektoncd_pruner_configuration_reloads_total[5m])
```

### Resource Age Distribution
```promql
histogram_quantile(0.5, rate(tektoncd_pruner_resource_age_at_deletion_seconds_bucket[5m]))
```

## Grafana Dashboard

A pre-configured Grafana dashboard is available at `examples/monitoring/grafana-dashboard.json`. This dashboard includes:

1. **Overview Panel**: Resource processing rates, error rates, and active resources
2. **Performance Panel**: Reconciliation duration, TTL processing time, and resource deletion time
3. **Error Analysis Panel**: Error breakdown by type and namespace
4. **Resource Age Panel**: Distribution of resource ages when deleted
5. **Configuration Panel**: Configuration reload events and errors
6. **Queue Status Panel**: Current queue depth and processing backlog

### Import Instructions

1. Open Grafana and navigate to **Dashboards** → **Import**
2. Upload the `examples/monitoring/grafana-dashboard.json` file
3. Select your Prometheus data source
4. Click **Import**

## Alerting

### Recommended Alerts

```yaml
groups:
  - name: tektoncd-pruner.rules
    rules:
      - alert: TektonPrunerDown
        expr: up{job="tekton-pruner"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Tekton Pruner is down"
          description: "Tekton Pruner has been down for more than 1 minute."

      - alert: TektonPrunerHighErrorRate
        expr: rate(tektoncd_pruner_resources_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate in Tekton Pruner"
          description: "Tekton Pruner error rate is {{ $value }} errors per second."

      - alert: TektonPrunerSlowProcessing
        expr: histogram_quantile(0.95, rate(tektoncd_pruner_reconciliation_duration_seconds_bucket[5m])) > 30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Tekton Pruner slow processing"
          description: "95th percentile reconciliation duration is {{ $value }} seconds."

      - alert: TektonPrunerHighQueueDepth
        expr: tektoncd_pruner_current_resources_queued > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High queue depth in Tekton Pruner"
          description: "Current queue depth is {{ $value }} resources."

      - alert: TektonPrunerConfigurationErrors
        expr: rate(tektoncd_pruner_configuration_errors_total[5m]) > 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Configuration errors in Tekton Pruner"
          description: "Configuration error rate is {{ $value }} errors per second."
```

## Troubleshooting

### Common Issues

#### 1. Metrics Not Available

**Symptoms**: Prometheus cannot scrape metrics, 404 errors on `/metrics`

**Solutions**:
```bash
# Check if metrics service exists
kubectl get svc tekton-pruner-controller-metrics -n tekton-pipelines

# Check if metrics are enabled
kubectl get configmap config-observability-tekton-pruner -n tekton-pipelines -o yaml

# Check controller logs
kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines

# Test metrics endpoint directly
kubectl port-forward svc/tekton-pruner-controller-metrics 9090:9090 -n tekton-pipelines
curl http://localhost:9090/metrics
```

#### 2. ServiceMonitor Not Working

**Symptoms**: ServiceMonitor exists but Prometheus Operator doesn't pick it up

**Solutions**:
```bash
# Check if Prometheus Operator is installed
kubectl get crd servicemonitors.monitoring.coreos.com

# Check ServiceMonitor labels and selectors
kubectl get servicemonitor tekton-pruner-controller -n tekton-pipelines -o yaml

# Check Prometheus Operator logs
kubectl logs -n monitoring-system deployment/prometheus-operator
```

#### 3. No Metrics Data in Prometheus

**Symptoms**: Prometheus shows target as UP but no metrics appear

**Solutions**:
```bash
# Check if observability is enabled in controller
kubectl get deployment tekton-pruner-controller -n tekton-pipelines -o yaml | grep -i metrics

# Check for configuration errors
kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines | grep -i observability

# Verify service endpoints
kubectl get endpoints tekton-pruner-controller-metrics -n tekton-pipelines
```

#### 4. Tracing Issues

**Symptoms**: No traces appearing in tracing backend

**Solutions**:
```bash
# Check tracing configuration
kubectl get configmap config-observability-tekton-pruner -n tekton-pipelines -o yaml | grep tracing

# Check OTLP endpoint connectivity
kubectl exec deployment/tekton-pruner-controller -n tekton-pipelines -- nc -zv otel-collector 4317

# Check controller logs for tracing errors
kubectl logs deployment/tekton-pruner-controller -n tekton-pipelines | grep -i trace
```

### Performance Impact

The observability features are designed to have minimal performance impact:

- **Metrics Collection**: ~1-2% CPU overhead
- **Memory Usage**: ~10-20MB additional memory
- **Network**: Minimal bandwidth usage for metrics scraping
- **Tracing**: Configurable sampling rate (default: 10%)

To reduce performance impact in high-throughput environments:

1. **Reduce metrics collection frequency**:
   ```yaml
   otel.metrics.interval: "60s"  # Increase from 30s
   ```

2. **Lower tracing sample rate**:
   ```yaml
   otel.tracing.sample-rate: "0.01"  # Decrease from 0.1
   ```

3. **Disable non-essential metrics**:
   ```yaml
   otel.metrics.enabled: "false"
   ```

## Integration Examples

### OpenTelemetry Collector

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    
    processors:
      batch:
    
    exporters:
      prometheus:
        endpoint: "0.0.0.0:8889"
      jaeger:
        endpoint: jaeger-collector:14250
        tls:
          insecure: true
    
    service:
      pipelines:
        metrics:
          receivers: [otlp]
          processors: [batch]
          exporters: [prometheus]
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [jaeger]
```

### Custom Metrics Backend

```yaml
env:
  - name: OTLP_METRICS_ENABLED
    value: "true"
  - name: OTLP_ENDPOINT
    value: "http://my-metrics-backend:4317"
  - name: OTLP_HEADERS
    value: "Authorization=Bearer token123"
```

## Best Practices

1. **Resource Labeling**: Use namespace and resource_type labels for effective filtering
2. **Alert Thresholds**: Set appropriate thresholds based on your environment
3. **Dashboard Organization**: Group related metrics for better observability
4. **Sampling Strategy**: Use appropriate sampling rates for tracing in production
5. **Monitoring the Monitor**: Set up alerts for the observability system itself

For additional examples and configuration options, see the `examples/monitoring/` directory. 