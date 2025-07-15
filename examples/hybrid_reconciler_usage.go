/*
Copyright 2025 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package examples demonstrates how to use the hybrid metrics approach
// This gives you the BEST of both worlds:
// ✅ Knative controller metrics (industry standard)
// ✅ Comprehensive pruner-specific metrics (detailed insights)
package examples

import (
	"context"
	"time"

	prunermetrics "github.com/openshift-pipelines/tektoncd-pruner/pkg/metrics"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

// ============================================================================
// What You Get: Complete Controller Observability Stack
// ============================================================================

/*
🎯 KNATIVE CONTROLLER METRICS (Standard Controller Observability):
   📊 reconcile_count (with tags: reconciler, success, namespace)
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

🎯 COMPREHENSIVE PRUNER METRICS (16+ Detailed Insights):
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
*/

// ============================================================================
// Example Reconciler with Hybrid Metrics
// ============================================================================

// ExampleReconciler shows how to use hybrid metrics in a reconciler
type ExampleReconciler struct {
	// Use the hybrid reporter for COMPLETE observability
	hybridReporter *prunermetrics.HybridReporter
}

// ReconcileKind demonstrates the hybrid metrics approach
func (r *ExampleReconciler) ReconcileKind(ctx context.Context, tr *pipelinev1.TaskRun) reconciler.Event {
	startTime := time.Now()
	logger := logging.FromContext(ctx)
	key := types.NamespacedName{Namespace: tr.Namespace, Name: tr.Name}

	// Initialize hybrid reporter (do this once per reconciler)
	if r.hybridReporter == nil {
		var err error
		r.hybridReporter, err = prunermetrics.NewHybridReporter("my-controller", logger)
		if err != nil {
			logger.Errorw("Failed to initialize hybrid metrics", "error", err)
			return err
		}
	}

	// This deferred call reports to BOTH metric systems:
	// - Knative: reconcile_count, reconcile_latency (with proper tags)
	// - OpenTelemetry: reconciliation_duration_seconds, resources_processed_total
	defer func() {
		duration := time.Since(startTime)
		r.hybridReporter.ReportReconcile(duration, true, key, "taskrun")
	}()

	// Report queue depth to Knative controller metrics
	r.hybridReporter.ReportQueueDepth(5) // Knative: work_queue_depth

	// Process the resource
	err := r.processTaskRun(ctx, tr)

	// Report detailed pruner-specific metrics
	if err == nil {
		r.hybridReporter.ReportResourceDeleted(tr.Namespace, "taskrun", "ttl_expired")
		r.hybridReporter.ReportTTLProcessingDuration(tr.Namespace, "taskrun", 100*time.Millisecond)
		r.hybridReporter.ReportResourceAgeAtDeletion(tr.Namespace, "taskrun", 24*time.Hour)
	} else {
		r.hybridReporter.ReportResourceError(tr.Namespace, "taskrun", "processing_failed")
	}

	logger.Infow("TaskRun processed with hybrid metrics")
	return err
}

func (r *ExampleReconciler) processTaskRun(ctx context.Context, tr *pipelinev1.TaskRun) error {
	// Your reconciliation logic here
	return nil
}

// ============================================================================
// Prometheus Queries You Can Run
// ============================================================================

/*
📊 KNATIVE CONTROLLER METRICS QUERIES:

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

# Kubernetes API performance
histogram_quantile(0.95, rate(client_latency_bucket[5m]))
rate(client_results[5m]) by (code)

📊 DETAILED PRUNER METRICS QUERIES:

# Resource processing rate by type
rate(tektoncd_pruner_resources_processed_total[5m]) by (namespace, resource_type, status)

# TTL processing efficiency
histogram_quantile(0.95, rate(tektoncd_pruner_ttl_processing_duration_seconds_bucket[5m]))

# Age of resources when deleted
histogram_quantile(0.50, rate(tektoncd_pruner_resource_age_at_deletion_seconds_bucket[5m]))

# Garbage collection performance
tektoncd_pruner_garbage_collection_duration_seconds
tektoncd_pruner_namespaces_processed_total

# Error breakdown
rate(tektoncd_pruner_resources_errors_total[5m]) by (namespace, resource_type, reason)

# Active worker utilization
tektoncd_pruner_active_workers_count
tektoncd_pruner_current_resources_queued
*/

// ============================================================================
// Usage Patterns
// ============================================================================

// Pattern 1: Pure Hybrid (Recommended) - Get everything!
func UseHybridApproach(logger *zap.SugaredLogger) {
	hybridReporter, err := prunermetrics.NewHybridReporter("my-controller", logger)
	if err != nil {
		logger.Fatalw("Failed to setup hybrid metrics", "error", err)
	}

	// This reports to BOTH Knative AND OpenTelemetry
	key := types.NamespacedName{Namespace: "default", Name: "test-taskrun"}
	hybridReporter.ReportReconcile(250*time.Millisecond, true, key, "taskrun")

	// Queue metrics go to Knative controller metrics
	hybridReporter.ReportQueueDepth(10)

	// Detailed metrics go to OpenTelemetry
	hybridReporter.ReportTTLProcessingDuration("default", "taskrun", 100*time.Millisecond)
	hybridReporter.ReportResourceAgeAtDeletion("default", "taskrun", 2*time.Hour)

	logger.Info("🎉 You now have COMPLETE observability!")
	logger.Info("📊 Knative metrics: reconcile_count, reconcile_latency, work_queue_depth, workqueue_*")
	logger.Info("🔍 Pruner metrics: 16+ comprehensive pruning insights")
}

// Pattern 2: OpenTelemetry Only (if you prefer single stack)
func UseOpenTelemetryOnly() {
	reporter := prunermetrics.GetReporter()
	if reporter != nil {
		reporter.ReportResourceProcessed("default", "taskrun", "success")
		reporter.ReportReconciliationDuration("default", "taskrun", 250*time.Millisecond)
	}
}

// Pattern 3: Knative Configuration Compatible (if needed)
func UseKnativeConfig(ctx context.Context, logger *zap.SugaredLogger) {
	// This respects Knative's config-observability but uses OpenTelemetry underneath
	err := prunermetrics.SetupWithKnativeConfig(ctx, logger, nil)
	if err != nil {
		logger.Fatalw("Failed to setup Knative-compatible metrics", "error", err)
	}

	// Still get all the benefits
	reporter := prunermetrics.GetReporter()
	if reporter != nil {
		reporter.ReportResourceProcessed("default", "taskrun", "success")
	}
}

// ============================================================================
// Benefits Summary
// ============================================================================

/*
✅ KNATIVE CONTROLLER METRICS BENEFITS:
   - Industry standard controller observability
   - Consistent with other Kubernetes controllers
   - Well-known metrics for SRE teams
   - Out-of-the-box Grafana dashboards available
   - Standardized latency buckets (10ms to 60s)
   - Automatic workqueue monitoring
   - Kubernetes API performance tracking

✅ COMPREHENSIVE PRUNER METRICS BENEFITS:
   - Detailed insights into pruning operations
   - TTL and history-based pruning analytics
   - Resource age tracking at deletion
   - Configuration change monitoring
   - Error breakdown by reason and type
   - Garbage collection performance
   - Worker utilization tracking

🏆 HYBRID APPROACH = BEST OF BOTH WORLDS!
   - Zero metric system migration needed
   - Complete controller observability
   - Detailed domain-specific insights
   - Production-ready monitoring
   - Future-proof architecture
*/
