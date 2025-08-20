# Tekton Pruner Metrics

The Tekton Pruner exposes OpenTelemetry metrics via Prometheus format on port 9090 at `/metrics`.

## Available Metrics

### Counters
- `tekton_pruner_controller_resources_processed` - Unique resources processed (deduplicated)
- `tekton_pruner_controller_resources_deleted` - Total resources deleted  
- `tekton_pruner_controller_resources_errors` - Total processing errors
- `tekton_pruner_controller_pipelineruns_processed` - Unique PipelineRuns processed (deduplicated)
- `tekton_pruner_controller_taskruns_processed` - Unique TaskRuns processed (deduplicated)
- `tekton_pruner_controller_pipelineruns_deleted` - Total PipelineRuns deleted
- `tekton_pruner_controller_taskruns_deleted` - Total TaskRuns deleted
- `tekton_pruner_controller_deletions_by_reason` - Total deletions by reason

### Histograms
- `tekton_pruner_controller_reconciliation_duration` - Reconciliation duration in seconds
- `tekton_pruner_controller_ttl_processing_duration` - TTL processing duration in seconds
- `tekton_pruner_controller_history_processing_duration` - History processing duration in seconds
- `tekton_pruner_controller_resource_age_at_deletion` - Resource age at deletion in seconds

### Gauges
- `tekton_pruner_controller_active_resources` - Current active resources count
- `tekton_pruner_controller_pending_deletions` - Current pending deletions count

## Common Labels
- `namespace` - Kubernetes namespace
- `resource_type` - `pipelinerun` or `taskrun`
- `operation` - `ttl` or `history`
- `status` - `success` or `error`
- `deletion_reason` - `ttl_expired` or `history_limits`

## Example Prometheus Queries

```promql
# Deletion rate by resource type
rate(tekton_pruner_controller_resources_deleted[5m])

# TTL vs history deletions
sum(rate(tekton_pruner_controller_deletions_by_reason[5m])) by (deletion_reason)

# Processing latency 95th percentile
histogram_quantile(0.95, rate(tekton_pruner_controller_reconciliation_duration_bucket[5m]))

# Active resources by namespace
tekton_pruner_controller_active_resources
```

## Additional Framework Metrics

The controller also exposes Knative framework metrics with `kn_` prefix:
- `kn_workqueue_*` - Workqueue metrics with hardcoded Knative scope
- `kn_k8s_client_*` - Kubernetes client metrics

Note: Framework metrics cannot be customized and use the scope `knative.dev/pkg/observability/metrics/k8s`.

## Deduplication

The `*_processed` metrics use deduplication to ensure each unique resource is counted only once, even when:
- Multiple workers process the same resource concurrently
- Resources are re-reconciled due to updates or retries
- Periodic resync events trigger multiple reconciliations

This provides accurate counts of unique resources processed rather than reconciliation events.
