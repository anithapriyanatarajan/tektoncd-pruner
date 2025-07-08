# Observability and Monitoring

Tektoncd-pruner provides comprehensive observability through OpenTelemetry metrics and distributed tracing. This document explains how to configure and use these features.

## Overview

The pruner exposes metrics in Prometheus format and optionally supports distributed tracing through OpenTelemetry Protocol (OTLP). This enables integration with popular monitoring and observability tools like:

- **Prometheus** + **Grafana** for metrics visualization
- **Jaeger** or **Zipkin** for distributed tracing
- **OpenTelemetry Collector** for telemetry processing
- **Cloud monitoring services** (AWS CloudWatch, Google Cloud Monitoring, etc.)

## Configuration

### Environment Variables

Configure observability through environment variables in the controller deployment:

```yaml
env:
  # Basic Configuration
  - name: OTEL_SERVICE_NAME
    value: tektoncd-pruner
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
    value: "false"           # Set to "true" to enable tracing
  - name: TRACING_SAMPLE_RATE
    value: "0.1"             # Sample 10% of traces
  
  # OTLP Configuration (for external collectors)
  - name: OTLP_METRICS_ENABLED
    value: "false"
  - name: OTLP_TRACE_ENABLED
    value: "false"
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://jaeger:4317"
  - name: OTEL_EXPORTER_OTLP_INSECURE
    value: "true"
```

### ConfigMap Configuration

Configure advanced settings through the `config-observability-tekton-pruner` ConfigMap:

```yaml
data:
  # OpenTelemetry Configuration
  otel.metrics.enabled: "true"
  otel.metrics.interval: "30s"
  otel.prometheus.enabled: "true"
  otel.tracing.enabled: "false"
  otel.tracing.sample-rate: "0.1"
  
  # Resource attributes
  otel.resource.attributes: "service.name=tektoncd-pruner,service.version=v1.0.0"
  
  # Custom histogram buckets
  otel.metrics.duration.buckets: "0.001,0.01,0.1,0.5,1.0,2.0,5.0,10.0,30.0"
  otel.metrics.age.buckets: "60,300,1800,3600,7200,21600,86400,604800,2592000"
```

## Metrics Reference

### Resource Processing Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_resources_processed_total` | Counter | Total resources processed | `namespace`, `resource_type`, `status`, `config_level` |
| `tektoncd_pruner_resources_deleted_total` | Counter | Total resources deleted | `namespace`, `resource_type`, `reason`, `config_level` |
| `tektoncd_pruner_resources_skipped_total` | Counter | Total resources skipped | `namespace`, `resource_type`, `reason` |
| `tektoncd_pruner_resources_errors_total` | Counter | Total processing errors | `namespace`, `resource_type`, `error_type` |

### Performance Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_reconciliation_duration_seconds` | Histogram | Reconciliation duration | `namespace`, `resource_type` |
| `tektoncd_pruner_ttl_processing_duration_seconds` | Histogram | TTL processing duration | `namespace`, `resource_type` |
| `tektoncd_pruner_history_processing_duration_seconds` | Histogram | History processing duration | `namespace`, `resource_type` |
| `tektoncd_pruner_resource_deletion_duration_seconds` | Histogram | Resource deletion duration | `namespace`, `resource_type` |

### State Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_active_resources` | Gauge | Currently active resources | `namespace`, `resource_type` |
| `tektoncd_pruner_current_resources_queued` | Gauge | Resources in processing queue | `namespace`, `resource_type` |
| `tektoncd_pruner_resource_age_at_deletion_seconds` | Histogram | Age of resources when deleted | `namespace`, `resource_type` |

### Configuration Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tektoncd_pruner_configuration_reloads_total` | Counter | Configuration reload events | - |
| `tektoncd_pruner_configuration_errors_total` | Counter | Configuration errors | `error_type` |

## Setting Up Monitoring

### 1. Prometheus + Grafana Setup

#### Deploy the metrics service:
```bash
kubectl apply -f config/metrics-service.yaml
```

#### Check if you have Prometheus Operator:
```bash
kubectl get crd servicemonitors.monitoring.coreos.com
```

#### For Prometheus Operator:
```bash
# Only if you have Prometheus Operator installed
kubectl apply -f config/optional/servicemonitor.yaml
```

#### For standard Prometheus setup:
Use the configuration from `examples/monitoring/prometheus-config.yaml`:
```bash
# Add the scrape_configs section to your prometheus.yml
# See examples/monitoring/prometheus-config.yaml for complete configuration
```

#### Manual Prometheus configuration:
```yaml
scrape_configs:
  - job_name: 'tekton-pruner'
    static_configs:
      - targets: ['tekton-pruner-controller-metrics.tekton-pipelines.svc.cluster.local:9090']
    metrics_path: /metrics
    scrape_interval: 30s
```

#### Import Grafana dashboard:
```bash
# Import the provided dashboard configuration
kubectl create configmap tekton-pruner-dashboard \
  --from-file=examples/monitoring/grafana-dashboard.json \
  -n monitoring
```

### 2. Jaeger Tracing Setup

Enable tracing and configure OTLP endpoint:

```yaml
env:
  - name: TRACING_ENABLED
    value: "true"
  - name: OTLP_TRACE_ENABLED
    value: "true"
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://jaeger-collector:14250"
```

### 3. OpenTelemetry Collector Setup

For advanced telemetry processing, use the OpenTelemetry Collector:

```yaml
# otel-collector-config.yaml
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

## Accessing Metrics

### Direct Metrics Endpoint

Access metrics directly from the controller:

```bash
# Port-forward to the controller
kubectl port-forward -n tekton-pipelines \
  deployment/tekton-pruner-controller 9090:9090

# Query metrics
curl http://localhost:9090/metrics
```

### Sample Queries

#### Resource processing rate:
```promql
rate(tektoncd_pruner_resources_processed_total[5m])
```

#### Average reconciliation duration:
```promql
histogram_quantile(0.95, 
  rate(tektoncd_pruner_reconciliation_duration_seconds_bucket[5m])
)
```

#### Error rate by type:
```promql
rate(tektoncd_pruner_resources_errors_total[5m])
```

#### Resources cleaned by reason:
```promql
sum by (reason) (tektoncd_pruner_resources_deleted_total)
```

## Troubleshooting

### Common Issues

1. **Metrics not showing up**
   - Check if `METRICS_ENABLED=true`
   - Verify the service is running: `kubectl get svc tekton-pruner-controller-metrics`
   - Check controller logs for observability setup errors

2. **High memory usage**
   - Reduce metrics collection interval
   - Increase histogram bucket ranges
   - Enable sampling for high-cardinality labels

3. **Traces not appearing**
   - Verify `TRACING_ENABLED=true` and `OTLP_TRACE_ENABLED=true`
   - Check OTLP endpoint connectivity
   - Increase sampling rate for testing

### Debug Commands

```bash
# Check observability configuration
kubectl get configmap config-observability-tekton-pruner -o yaml

# View controller logs
kubectl logs -n tekton-pipelines deployment/tekton-pruner-controller

# Test metrics endpoint
kubectl exec -n tekton-pipelines deployment/tekton-pruner-controller -- \
  curl -s http://localhost:9090/metrics | grep tektoncd_pruner
```

## Performance Impact

The observability features have minimal performance impact:

- **Metrics collection**: ~1-2% CPU overhead
- **Tracing (10% sampling)**: ~0.5% CPU overhead  
- **Memory usage**: ~10-20MB additional memory

For high-throughput environments, consider:
- Reducing metrics collection interval
- Lowering tracing sample rate
- Using external metric aggregation 