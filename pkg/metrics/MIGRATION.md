# Native OpenTelemetry Implementation with Knative Controller Metrics

## Overview

The tektoncd-pruner observability implementation uses a **hybrid approach** that combines **Knative's excellent controller metrics** with **comprehensive OpenTelemetry metrics**. This gives you the best of both worlds without any migration overhead.

## Why Hybrid Approach?

Since this is a first-time observability implementation for tektoncd-pruner, we chose to implement both Knative controller metrics (industry standard) and OpenTelemetry natively rather than using the deprecated OpenCensus approach. This provides:

### ✅ **Knative Controller Metrics Benefits**
- **Industry Standard**: Consistent with other Kubernetes controllers
- **Well-Known Metrics**: SRE teams already know these metrics  
- **Proven Patterns**: `reconcile_count`, `reconcile_latency`, `work_queue_depth`
- **Automatic Workqueue Monitoring**: Built-in queue performance tracking
- **Kubernetes API Metrics**: Client latency and result tracking
- **Standardized Buckets**: Latency histograms with proven boundaries

### ✅ **Comprehensive OpenTelemetry Benefits**
- **Future-Proof**: No migration needed - already using the modern standard
- **Detailed Insights**: 16+ pruner-specific metrics for deep observability
- **Active Development**: OpenTelemetry is actively maintained and evolving
- **Vendor Agnostic**: Support for multiple backends without vendor lock-in
- **Better Performance**: Optimized resource usage and lower overhead
- **Rich Ecosystem**: Extensive integrations and tooling

### ✅ **Combined Benefits**
- **Complete Coverage**: Both standard controller AND domain-specific metrics
- **Zero Migration**: Start with the best stack from day one
- **No Technical Debt**: Clean implementation without legacy code paths

## Architecture

### Current Implementation (Hybrid: Knative + OpenTelemetry)

```go
// Knative Controller Metrics (via controller.StatsReporter)
import "knative.dev/pkg/controller"

// OpenTelemetry Metrics (native implementation)
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/exporters/prometheus"
)

// Hybrid reporter combines both systems
type HybridReporter struct {
    controllerStats controller.StatsReporter  // Knative metrics
    prunerReporter  *Reporter                 // OpenTelemetry metrics
}

// Reports to BOTH systems simultaneously
func (h *HybridReporter) ReportReconcile(duration time.Duration, success bool, key types.NamespacedName, resourceType string) {
    // Knative: reconcile_count, reconcile_latency (with proper tags)
    h.controllerStats.ReportReconcile(duration, successStr, key)
    
    // OpenTelemetry: detailed pruner metrics
    h.prunerReporter.ReportReconciliationDuration(key.Namespace, resourceType, duration)
    h.prunerReporter.ReportResourceProcessed(key.Namespace, resourceType, status)
}
```

### Metrics Implementation

| System | Type | Usage | Example |
|---|---|---|---|
| **Knative Controller** | Standard | Controller observability | `reconcile_count`, `work_queue_depth` |
| **OpenTelemetry Counter** | Cumulative | Resource tracking | `resources_processed_total` |
| **OpenTelemetry Histogram** | Distribution | Performance metrics | `reconciliation_duration_seconds` |
| **OpenTelemetry Gauge** | Current value | State metrics | `current_resources_queued` |

## Complete Metrics Stack

### 📊 **Knative Controller Metrics** (12 metrics)

| Metric | Type | Description | Tags |
|--------|------|-------------|------|
| `reconcile_count` | Counter | Reconcile operations | reconciler, success, namespace |
| `reconcile_latency` | Histogram | Reconcile duration | reconciler, success, namespace |
| `work_queue_depth` | Gauge | Current queue depth | reconciler |
| `workqueue_adds_total` | Counter | Items added to queue | name |
| `workqueue_depth` | Gauge | Current workqueue depth | name |
| `workqueue_queue_latency_seconds` | Histogram | Time in queue | name |
| `workqueue_retries_total` | Counter | Retry operations | name |
| `workqueue_work_duration_seconds` | Histogram | Processing time | name |
| `workqueue_unfinished_work_seconds` | Gauge | Outstanding work | name |
| `workqueue_longest_running_processor_seconds` | Gauge | Longest running item | name |
| `client_latency` | Histogram | K8s API latency | verb, host |
| `client_results` | Counter | API results | verb, host, code |

### 🔍 **Comprehensive Pruner Metrics** (16+ metrics)

All the detailed metrics we implemented:

- **Resource Processing** (4): `resources_processed_total`, `resources_deleted_total`, `resources_errors_total`, `resources_skipped_total`
- **Performance** (4): `reconciliation_duration_seconds`, `ttl_processing_duration_seconds`, `history_processing_duration_seconds`, `resource_deletion_duration_seconds`  
- **State** (3): `resources_queued_total`, `current_resources_queued`, `active_resources_count`
- **TTL Operations** (2): `ttl_annotation_updates_total`, `ttl_expiration_events_total`
- **History Limits** (2): `history_limit_events_total`, `resources_cleaned_by_history`
- **Configuration** (2): `configuration_reloads_total`, `configuration_errors_total`
- **Resource Age** (1): `resource_age_at_deletion_seconds`
- **Error Breakdown** (2): `resource_delete_errors_total`, `resource_update_errors_total`
- **Operational** (3): `garbage_collection_duration_seconds`, `namespaces_processed_total`, `active_workers_count`

## Usage Examples

### Hybrid Metrics Reporting
```go
// Initialize hybrid reporter (gets you EVERYTHING!)
hybridReporter, err := prunermetrics.NewHybridReporter("my-controller", logger)
if err != nil {
    logger.Fatalw("Failed to setup hybrid metrics", "error", err)
}

// Reports to BOTH Knative controller metrics AND OpenTelemetry
key := types.NamespacedName{Namespace: "default", Name: "my-resource"}
hybridReporter.ReportReconcile(250*time.Millisecond, true, key, "taskrun")

// Knative controller metric: work_queue_depth
hybridReporter.ReportQueueDepth(10)

// OpenTelemetry detailed metrics
hybridReporter.ReportTTLProcessingDuration("default", "taskrun", 100*time.Millisecond)
hybridReporter.ReportResourceAgeAtDeletion("default", "taskrun", 2*time.Hour)
```

### Distributed Tracing (OpenTelemetry)
```go
tracer := prunermetrics.GetTracer()

// Create trace spans
ctx, span := tracer.TraceReconcile(ctx, "taskrun", namespace, name)
defer tracer.EndSpan(span)

// Add events and attributes
tracer.AddAnnotation(ctx, "Resource processed", map[string]interface{}{
    "result": "success",
    "duration_ms": 250,
})
```

## Prometheus Queries

### Knative Controller Metrics
```promql
# Controller throughput by reconciler
rate(reconcile_count[5m]) by (reconciler)

# Controller error rate
rate(reconcile_count{success="false"}[5m]) / rate(reconcile_count[5m])

# Controller latency percentiles
histogram_quantile(0.95, rate(reconcile_latency_bucket[5m])) by (reconciler)

# Work queue depth across controllers
work_queue_depth by (reconciler)

# Workqueue performance
rate(workqueue_adds_total[5m])
histogram_quantile(0.50, rate(workqueue_queue_latency_seconds_bucket[5m]))
```

### Detailed Pruner Metrics
```promql
# Resource processing rate by type
rate(tektoncd_pruner_resources_processed_total[5m]) by (namespace, resource_type)

# TTL processing efficiency
histogram_quantile(0.95, rate(tektoncd_pruner_ttl_processing_duration_seconds_bucket[5m]))

# Resource age at deletion
histogram_quantile(0.50, rate(tektoncd_pruner_resource_age_at_deletion_seconds_bucket[5m]))
```

## Benefits Achieved

### ✅ **No Migration Needed**
- Started with modern observability stack
- Combined industry standards with detailed insights
- No legacy code or technical debt
- Clean, maintainable codebase

### ✅ **Complete Observability**
- Standard controller metrics (familiar to SRE teams)
- Comprehensive domain-specific metrics
- Distributed tracing capabilities
- Professional monitoring stack

### ✅ **Enterprise Ready**
- Production-grade metrics from day one
- Proven patterns plus detailed insights
- Compatible with existing monitoring infrastructure
- Future-proof architecture

### ✅ **Performance Optimized**
- Minimal overhead from both systems
- Efficient metric collection
- Optimized for Kubernetes environments
- Smart metric cardinality management

This hybrid approach provides the strongest possible foundation for observability that serves both immediate operational needs and long-term monitoring requirements. 