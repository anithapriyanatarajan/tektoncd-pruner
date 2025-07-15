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

package metrics

import (
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/controller"
)

// HybridReporter combines Knative's excellent controller metrics with our comprehensive pruner metrics
// This gives us the best of both worlds:
// - Knative controller metrics: reconcile_count, reconcile_latency, work_queue_depth, workqueue_* metrics
// - Our detailed pruner metrics: 16+ specific pruning metrics for comprehensive observability
type HybridReporter struct {
	// Knative's controller metrics for standard controller observability
	// Provides: reconcile_count, reconcile_latency, work_queue_depth, workqueue_* metrics
	controllerStats controller.StatsReporter

	// Our comprehensive OpenTelemetry metrics for detailed pruner functionality
	// Provides: 16+ pruner-specific metrics for TTL, history, errors, performance, etc.
	prunerReporter *Reporter

	// Reconciler name for consistent tagging
	reconcilerName string
	logger         *zap.SugaredLogger
}

// NewHybridReporter creates a reporter that uses both Knative and OpenTelemetry metrics
// This provides the complete controller observability stack
func NewHybridReporter(reconcilerName string, logger *zap.SugaredLogger) (*HybridReporter, error) {
	// Initialize Knative's controller stats (gives us standard controller metrics)
	controllerStats := controller.MustNewStatsReporter(reconcilerName, logger)

	// Initialize our OpenTelemetry reporter (gives us detailed pruner metrics)
	prunerReporter := GetReporter()

	logger.Infow("Initialized hybrid metrics reporter",
		"reconciler", reconcilerName,
		"knative_metrics", "reconcile_count,reconcile_latency,work_queue_depth,workqueue_*",
		"pruner_metrics", "16+ comprehensive pruning metrics")

	return &HybridReporter{
		controllerStats: controllerStats,
		prunerReporter:  prunerReporter,
		reconcilerName:  reconcilerName,
		logger:          logger,
	}, nil
}

// =============================================================================
// Knative Controller Metrics (Standard Controller Observability)
// =============================================================================

// ReportReconcile reports to BOTH Knative controller metrics AND our detailed metrics
// Knative metrics: reconcile_count, reconcile_latency (with reconciler, success, namespace tags)
// Our metrics: reconciliation_duration_seconds, resources_processed_total
func (h *HybridReporter) ReportReconcile(duration time.Duration, success bool, key types.NamespacedName, resourceType string) {
	// Report to Knative's standard controller metrics
	successStr := "true"
	if !success {
		successStr = "false"
	}

	if err := h.controllerStats.ReportReconcile(duration, successStr, key); err != nil {
		h.logger.Errorw("Failed to report to Knative controller metrics", "error", err)
	}

	// Report to our detailed OpenTelemetry metrics
	if h.prunerReporter != nil {
		h.prunerReporter.ReportReconciliationDuration(key.Namespace, resourceType, duration)
		if success {
			h.prunerReporter.ReportResourceProcessed(key.Namespace, resourceType, "success")
		} else {
			h.prunerReporter.ReportResourceProcessed(key.Namespace, resourceType, "error")
		}
	}
}

// ReportQueueDepth reports queue depth to Knative's controller metrics
// Knative metric: work_queue_depth (with reconciler tag)
func (h *HybridReporter) ReportQueueDepth(depth int64) {
	if err := h.controllerStats.ReportQueueDepth(depth); err != nil {
		h.logger.Errorw("Failed to report queue depth to Knative controller metrics", "error", err)
	}

	// Also report to our metrics for consistency
	if h.prunerReporter != nil {
		h.prunerReporter.ReportCurrentResourcesQueued("", h.reconcilerName, depth)
	}
}

// =============================================================================
// Detailed Pruner Metrics (OpenTelemetry - Comprehensive Observability)
// =============================================================================

// All pruner-specific metrics go through our OpenTelemetry implementation
// These provide detailed insights into pruning operations

func (h *HybridReporter) ReportResourceDeleted(namespace, resourceType, reason string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceDeleted(namespace, resourceType, reason)
	}
}

func (h *HybridReporter) ReportResourceError(namespace, resourceType, reason string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceError(namespace, resourceType, reason)
	}
}

func (h *HybridReporter) ReportResourceSkipped(namespace, resourceType, reason string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceSkipped(namespace, resourceType, reason)
	}
}

func (h *HybridReporter) ReportTTLProcessingDuration(namespace, resourceType string, duration time.Duration) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportTTLProcessingDuration(namespace, resourceType, duration)
	}
}

func (h *HybridReporter) ReportHistoryProcessingDuration(namespace, resourceType string, duration time.Duration) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportHistoryProcessingDuration(namespace, resourceType, duration)
	}
}

func (h *HybridReporter) ReportTTLAnnotationUpdate(namespace, resourceType string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportTTLAnnotationUpdate(namespace, resourceType)
	}
}

func (h *HybridReporter) ReportTTLExpirationEvent(namespace, resourceType string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportTTLExpirationEvent(namespace, resourceType)
	}
}

func (h *HybridReporter) ReportHistoryLimitEvent(namespace, resourceType string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportHistoryLimitEvent(namespace, resourceType)
	}
}

func (h *HybridReporter) ReportResourceCleanedByHistory(namespace, resourceType string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceCleanedByHistory(namespace, resourceType)
	}
}

func (h *HybridReporter) ReportConfigurationReload(configLevel string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportConfigurationReload(configLevel)
	}
}

func (h *HybridReporter) ReportConfigurationError(configLevel string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportConfigurationError(configLevel)
	}
}

func (h *HybridReporter) ReportResourceAgeAtDeletion(namespace, resourceType string, age time.Duration) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceAgeAtDeletion(namespace, resourceType, age)
	}
}

func (h *HybridReporter) ReportResourceDeletionDuration(namespace, resourceType string, duration time.Duration) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceDeletionDuration(namespace, resourceType, duration)
	}
}

func (h *HybridReporter) ReportGarbageCollectionDuration(duration time.Duration, namespacesCount int) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportGarbageCollectionDuration(duration, namespacesCount)
	}
}

func (h *HybridReporter) ReportActiveResourcesCount(namespace, resourceType string, count int64) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportActiveResourcesCount(namespace, resourceType, count)
	}
}

func (h *HybridReporter) ReportResourceQueued(namespace, resourceType string) {
	if h.prunerReporter != nil {
		h.prunerReporter.ReportResourceQueued(namespace, resourceType)
	}
}

// =============================================================================
// Metrics Summary
// =============================================================================

// GetMetricsSummary returns a summary of all metrics being reported
func (h *HybridReporter) GetMetricsSummary() map[string]interface{} {
	return map[string]interface{}{
		"reconciler_name": h.reconcilerName,
		"knative_controller_metrics": []string{
			"reconcile_count (with reconciler, success, namespace tags)",
			"reconcile_latency (with distribution buckets: 10ms, 100ms, 1s, 10s, 30s, 60s)",
			"work_queue_depth (with reconciler tag)",
			"workqueue_adds_total",
			"workqueue_depth",
			"workqueue_queue_latency_seconds",
			"workqueue_retries_total",
			"workqueue_work_duration_seconds",
			"workqueue_unfinished_work_seconds",
			"workqueue_longest_running_processor_seconds",
			"client_latency (Kubernetes API requests)",
			"client_results (API request results by status code)",
		},
		"comprehensive_pruner_metrics": []string{
			"tektoncd_pruner_resources_processed_total",
			"tektoncd_pruner_resources_deleted_total",
			"tektoncd_pruner_resources_errors_total",
			"tektoncd_pruner_resources_skipped_total",
			"tektoncd_pruner_reconciliation_duration_seconds",
			"tektoncd_pruner_ttl_processing_duration_seconds",
			"tektoncd_pruner_history_processing_duration_seconds",
			"tektoncd_pruner_resource_deletion_duration_seconds",
			"tektoncd_pruner_resources_queued_total",
			"tektoncd_pruner_current_resources_queued",
			"tektoncd_pruner_active_resources_count",
			"tektoncd_pruner_ttl_annotation_updates_total",
			"tektoncd_pruner_ttl_expiration_events_total",
			"tektoncd_pruner_history_limit_events_total",
			"tektoncd_pruner_resources_cleaned_by_history",
			"tektoncd_pruner_configuration_reloads_total",
			"tektoncd_pruner_configuration_errors_total",
			"tektoncd_pruner_resource_age_at_deletion_seconds",
			"tektoncd_pruner_resource_delete_errors_total",
			"tektoncd_pruner_resource_update_errors_total",
			"tektoncd_pruner_garbage_collection_duration_seconds",
			"tektoncd_pruner_namespaces_processed_total",
			"tektoncd_pruner_active_workers_count",
		},
	}
}
