# Tektoncd-Pruner Observability

This package provides **complete controller observability** by combining Knative's excellent controller metrics with comprehensive pruner-specific insights, following OpenTelemetry standards.

## 🏆 Hybrid Observability: Best of Both Worlds

Our **hybrid approach** gives you the complete observability stack:

### ✅ **Knative Controller Metrics** (Industry Standard)
```
📊 reconcile_count (with reconciler, success, namespace tags)
📈 reconcile_latency (histogram: 10ms, 100ms, 1s, 10s, 30s, 60s)  
🔢 work_queue_depth (current queue depth with reconciler tag)
⚡ workqueue_adds_total (total items added to queue)
📏 workqueue_depth (current workqueue depth)
⏱️  workqueue_queue_latency_seconds (time items wait in queue)
🔄 workqueue_retries_total (total retry operations)
⚙️  workqueue_work_duration_seconds (processing time per item)
📊 workqueue_unfinished_work_seconds (outstanding work time)  
⏰ workqueue_longest_running_processor_seconds (longest running item)
🌐 client_latency (Kubernetes API request latency)
📝 client_results (API request results by status code)
```

### ✅ **Comprehensive Pruner Metrics** (16+ Detailed Insights)
```
🗑️  tektoncd_pruner_resources_processed_total
🧹 tektoncd_pruner_resources_deleted_total
❌ tektoncd_pruner_resources_errors_total
⏭️  tektoncd_pruner_resources_skipped_total
⚡ tektoncd_pruner_reconciliation_duration_seconds
⏰ tektoncd_pruner_ttl_processing_duration_seconds  
📚 tektoncd_pruner_history_processing_duration_seconds
🗂️  tektoncd_pruner_resource_deletion_duration_seconds
📥 tektoncd_pruner_resources_queued_total
📊 tektoncd_pruner_current_resources_queued
🏃 tektoncd_pruner_active_resources_count
🏷️  tektoncd_pruner_ttl_annotation_updates_total
⌛ tektoncd_pruner_ttl_expiration_events_total
📋 tektoncd_pruner_history_limit_events_total
🧽 tektoncd_pruner_resources_cleaned_by_history
⚙️  tektoncd_pruner_configuration_reloads_total
⚠️  tektoncd_pruner_configuration_errors_total
📅 tektoncd_pruner_resource_age_at_deletion_seconds
💥 tektoncd_pruner_resource_delete_errors_total
🔧 tektoncd_pruner_resource_update_errors_total
🗑️  tektoncd_pruner_garbage_collection_duration_seconds
🏢 tektoncd_pruner_namespaces_processed_total
👷 tektoncd_pruner_active_workers_count
```

## Quick Start: Hybrid Metrics

```go
import prunermetrics "github.com/openshift-pipelines/tektoncd-pruner/pkg/metrics"

// Initialize hybrid reporter (gets you EVERYTHING!)
hybridReporter, err := prunermetrics.NewHybridReporter("my-controller", logger)
if err != nil {
    logger.Fatalw("Failed to setup hybrid metrics", "error", err)
}

// This reports to BOTH Knative AND OpenTelemetry metrics systems
key := types.NamespacedName{Namespace: "default", Name: "my-resource"}
hybridReporter.ReportReconcile(250*time.Millisecond, true, key, "taskrun")

// Queue metrics go to Knative (work_queue_depth)
hybridReporter.ReportQueueDepth(10)

// Detailed insights go to OpenTelemetry
hybridReporter.ReportTTLProcessingDuration("default", "taskrun", 100*time.Millisecond)
hybridReporter.ReportResourceAgeAtDeletion("default", "taskrun", 2*time.Hour)
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Reconcilers   │───▶│  Hybrid Reporter │───▶│   Prometheus    │
│                 │    │                  │    │                 │
│ - TaskRun       │    │ ┌──────────────┐ │    │ - /metrics      │
│ - PipelineRun   │    │ │ Knative      │ │    │ - Scraping      │
│ - TektonPruner  │    │ │ Controller   │ │    │ - Alerting      │
└─────────────────┘    │ │ Metrics      │ │    └─────────────────┘
                       │ └──────────────┘ │
┌─────────────────┐    │ ┌──────────────┐ │    ┌─────────────────┐
│   Distributed   │───▶│ │ OpenTelemetry│ │───▶│   Jaeger        │
│   Tracing       │    │ │ 16+ Detailed │ │    │                 │
│                 │    │ │ Metrics      │ │    │ - Span Traces   │
│ - Operations    │    │ └──────────────┘ │    │ - Dependencies  │
│ - Performance   │    └──────────────────┘    │ - Performance   │
└─────────────────┘                            └─────────────────┘
```

## Implemented Metrics

### 📊 Resource Processing Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_resources_processed_total` | Counter | Total resources processed | namespace, resource_type, status |
| `tektoncd_pruner_resources_deleted_total` | Counter | Total resources deleted | namespace, resource_type, reason |
| `tektoncd_pruner_resources_errors_total` | Counter | Total processing errors | namespace, resource_type, reason |
| `tektoncd_pruner_resources_skipped_total` | Counter | Total resources skipped | namespace, resource_type, reason |

### ⚡ Performance Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_reconciliation_duration_seconds` | Histogram | Time spent in reconciliation | namespace, resource_type |
| `tektoncd_pruner_ttl_processing_duration_seconds` | Histogram | Time spent processing TTL | namespace, resource_type |
| `tektoncd_pruner_history_processing_duration_seconds` | Histogram | Time spent processing history limits | namespace, resource_type |
| `tektoncd_pruner_resource_deletion_duration_seconds` | Histogram | Time spent deleting resources | namespace, resource_type |

### 📈 State Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_resources_queued_total` | Counter | Total resources queued | namespace, resource_type |
| `tektoncd_pruner_current_resources_queued` | Gauge | Current resources in queue | namespace, resource_type |
| `tektoncd_pruner_active_resources_count` | Gauge | Current active resources | namespace, resource_type |

### ⏰ TTL-specific Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_ttl_annotation_updates_total` | Counter | TTL annotation updates | namespace, resource_type |
| `tektoncd_pruner_ttl_expiration_events_total` | Counter | TTL expiration events | namespace, resource_type |

### 📚 History Limit Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_history_limit_events_total` | Counter | History limit events | namespace, resource_type |
| `tektoncd_pruner_resources_cleaned_by_history` | Counter | Resources cleaned by history limits | namespace, resource_type |

### ⚙️ Configuration Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_configuration_reloads_total` | Counter | Configuration reloads | config_level |
| `tektoncd_pruner_configuration_errors_total` | Counter | Configuration errors | config_level |

### 📅 Resource Age Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_resource_age_at_deletion_seconds` | Histogram | Age of resources when deleted | namespace, resource_type |

### 🚨 Error Breakdown Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_resource_delete_errors_total` | Counter | Resource deletion errors | namespace, resource_type |
| `tektoncd_pruner_resource_update_errors_total` | Counter | Resource update errors | namespace, resource_type |

### 🏗️ Operational Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|---------|
| `tektoncd_pruner_garbage_collection_duration_seconds` | Histogram | Complete GC cycle duration | - |
| `tektoncd_pruner_namespaces_processed_total` | Counter | Namespaces processed during GC | - |
| `tektoncd_pruner_active_workers_count` | Gauge | Active worker goroutines | - |

## Usage Examples

### Basic Metrics Reporting

```go
reporter := prunermetrics.GetReporter()

// Report resource processing
reporter.ReportResourceProcessed("default", "taskrun", "success")

// Report performance metrics
reporter.ReportReconciliationDuration("default", "taskrun", 250*time.Millisecond)

// Report errors
reporter.ReportResourceError("default", "pipelinerun", "ttl_processing_failed")
```

### Tracing Operations

```go
tracer := prunermetrics.GetTracer()

// Trace reconciliation
ctx, span := tracer.TraceReconcile(ctx, "taskrun", namespace, name)
defer tracer.EndSpan(span)

// Add annotations
tracer.AddAnnotation(ctx, "Resource processed", map[string]interface{}{
    "result": "success",
    "duration_ms": 250,
})
```

### Integration in Reconcilers

```go
func (r *Reconciler) ReconcileKind(ctx context.Context, tr *pipelinev1.TaskRun) reconciler.Event {
    startTime := time.Now()
    reporter := prunermetrics.GetReporter()
    
    defer func() {
        duration := time.Since(startTime)
        reporter.ReportReconciliationDuration(tr.Namespace, "taskrun", duration)
    }()
    
    // Report processing start
    reporter.ReportResourceProcessed(tr.Namespace, "taskrun", "processing")
    
    // Your reconciliation logic here...
    
    // Report success
    reporter.ReportResourceProcessed(tr.Namespace, "taskrun", "success")
    return nil
}
```

## Prometheus Query Examples

### Resource Processing Rate
```promql
# Resources processed per second
rate(tektoncd_pruner_resources_processed_total[5m])

# Error rate by namespace
rate(tektoncd_pruner_resources_errors_total[5m]) / rate(tektoncd_pruner_resources_processed_total[5m])
```

### Performance Analysis
```promql
# 95th percentile reconciliation latency
histogram_quantile(0.95, rate(tektoncd_pruner_reconciliation_duration_seconds_bucket[5m]))

# TTL processing latency by namespace
histogram_quantile(0.50, rate(tektoncd_pruner_ttl_processing_duration_seconds_bucket[5m])) by (namespace)
```

### Operational Monitoring
```promql
# Current queue depth
tektoncd_pruner_current_resources_queued

# Active workers
tektoncd_pruner_active_workers_count

# Configuration reload frequency
rate(tektoncd_pruner_configuration_reloads_total[1h])
```

## Native OpenTelemetry Implementation

✅ **Modern Stack**: This implementation uses **OpenTelemetry natively** - no migration needed! See [MIGRATION.md](./MIGRATION.md) for implementation details.

### Current Architecture (OpenTelemetry Native)
- Uses `go.opentelemetry.io/otel/metric` for metrics
- Uses `go.opentelemetry.io/otel/trace` for tracing  
- Direct Prometheus integration via OpenTelemetry exporter
- Future-proof and actively maintained

## Configuration

### Observability Config (config-observability.yaml)
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-observability-tekton-pruner
  namespace: tekton-pipelines
data:
  metrics.backend-destination: prometheus
  metrics.request-metrics-backend-destination: prometheus
```

### Metric Buckets
- **Latency Metrics**: 1ms to 10 minutes (125-series buckets)
- **Age Metrics**: 1 minute to 30 days (logarithmic buckets)

## Monitoring Setup

### Prometheus Configuration
```yaml
- job_name: 'tekton-pruner'
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app]
    action: keep
    regex: tekton-pruner-controller
  - source_labels: [__meta_kubernetes_pod_container_port_name]
    action: keep
    regex: metrics
```

### Grafana Dashboard Queries
See [monitoring/](../monitoring/) directory for complete Grafana dashboard configurations.

## Testing

### Metrics Validation
```bash
# Check metrics endpoint
curl http://tekton-pruner-controller:9090/metrics | grep tektoncd_pruner

# Validate metric types
curl -s http://tekton-pruner-controller:9090/metrics | promtool check metrics
```

### Load Testing
```bash
# Generate load and monitor metrics
kubectl apply -f test/load/pipelineruns.yaml
kubectl port-forward svc/tekton-pruner-controller 9090:9090
```

## Best Practices

1. **Label Cardinality**: Keep labels bounded (avoid high-cardinality values like timestamps)
2. **Metric Naming**: Follow Prometheus naming conventions (use descriptive names, avoid abbreviations)
3. **Performance**: Metrics collection has minimal performance impact (<1% CPU overhead)
4. **Alerting**: Set up alerts on error rates and processing latencies
5. **Retention**: Configure appropriate metric retention policies based on storage capacity

## Troubleshooting

### Common Issues
- **Missing Metrics**: Check if observability is enabled in controller configuration
- **High Cardinality**: Monitor namespace and resource_type label usage
- **Performance**: Use sampling for high-frequency operations if needed

### Debug Mode
```yaml
# Enable debug logging for metrics
data:
  loglevel.controller: "debug"
  metrics.debug: "true"
```

## Contributing

When adding new metrics:
1. Follow the naming convention: `tektoncd_pruner_<component>_<metric>_<unit>`
2. Add appropriate labels for filtering and aggregation
3. Update this documentation with examples
4. Add corresponding Grafana dashboard panels
5. Include unit tests for new metrics

For more details, see [MIGRATION.md](./MIGRATION.md) and the [OpenTelemetry migration plan](https://opentelemetry.io/blog/2023/sunsetting-opencensus/). 