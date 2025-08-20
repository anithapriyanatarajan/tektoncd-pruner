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
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	// Metric names
	MetricResourcesProcessed        = "tekton_pruner_controller_resources_processed"
	MetricResourcesDeleted          = "tekton_pruner_controller_resources_deleted"
	MetricResourcesErrors           = "tekton_pruner_controller_resources_errors"
	MetricReconciliationDuration    = "tekton_pruner_controller_reconciliation_duration"
	MetricTTLProcessingDuration     = "tekton_pruner_controller_ttl_processing_duration"
	MetricHistoryProcessingDuration = "tekton_pruner_controller_history_processing_duration"
	MetricActiveResourcesCount      = "tekton_pruner_controller_active_resources"
	MetricPendingDeletionsCount     = "tekton_pruner_controller_pending_deletions"
	MetricResourceAgeAtDeletion     = "tekton_pruner_controller_resource_age_at_deletion"
	MetricPipelineRunsProcessed     = "tekton_pruner_controller_pipelineruns_processed"
	MetricTaskRunsProcessed         = "tekton_pruner_controller_taskruns_processed"
	MetricPipelineRunsDeleted       = "tekton_pruner_controller_pipelineruns_deleted"
	MetricTaskRunsDeleted           = "tekton_pruner_controller_taskruns_deleted"
	MetricDeletionsByReason         = "tekton_pruner_controller_deletions_by_reason"

	// Resource types
	ResourceTypePipelineRun = "pipelinerun"
	ResourceTypeTaskRun     = "taskrun"

	// Operations
	OperationTTL     = "ttl"
	OperationHistory = "history"

	// Status values
	StatusSuccess = "success"
	StatusError   = "error"

	// Deletion reasons
	DeletionReasonTTL    = "ttl_expired"
	DeletionReasonLimits = "history_limits"
)

type Recorder struct {
	resourcesProcessed        metric.Int64Counter
	resourcesDeleted          metric.Int64Counter
	resourcesErrors           metric.Int64Counter
	pipelineRunsProcessed     metric.Int64Counter
	taskRunsProcessed         metric.Int64Counter
	pipelineRunsDeleted       metric.Int64Counter
	taskRunsDeleted           metric.Int64Counter
	deletionsByReason         metric.Int64Counter
	reconciliationDuration    metric.Float64Histogram
	ttlProcessingDuration     metric.Float64Histogram
	historyProcessingDuration metric.Float64Histogram
	resourceAgeAtDeletion     metric.Float64Histogram
	activeResourcesCount      metric.Int64UpDownCounter
	pendingDeletionsCount     metric.Int64UpDownCounter
}

var (
	recorder *Recorder
	once     sync.Once
)

func GetRecorder() *Recorder {
	once.Do(func() {
		recorder = newRecorder()
	})
	return recorder
}

func newRecorder() *Recorder {
	meter := otel.Meter("tekton-pruner-controller")
	r := &Recorder{}

	r.resourcesProcessed, _ = meter.Int64Counter(MetricResourcesProcessed, metric.WithUnit("1"))
	r.resourcesDeleted, _ = meter.Int64Counter(MetricResourcesDeleted, metric.WithUnit("1"))
	r.resourcesErrors, _ = meter.Int64Counter(MetricResourcesErrors, metric.WithUnit("1"))
	r.pipelineRunsProcessed, _ = meter.Int64Counter(MetricPipelineRunsProcessed, metric.WithUnit("1"))
	r.taskRunsProcessed, _ = meter.Int64Counter(MetricTaskRunsProcessed, metric.WithUnit("1"))
	r.pipelineRunsDeleted, _ = meter.Int64Counter(MetricPipelineRunsDeleted, metric.WithUnit("1"))
	r.taskRunsDeleted, _ = meter.Int64Counter(MetricTaskRunsDeleted, metric.WithUnit("1"))
	r.deletionsByReason, _ = meter.Int64Counter(MetricDeletionsByReason, metric.WithUnit("1"))

	r.reconciliationDuration, _ = meter.Float64Histogram(MetricReconciliationDuration, metric.WithUnit("s"))
	r.ttlProcessingDuration, _ = meter.Float64Histogram(MetricTTLProcessingDuration, metric.WithUnit("s"))
	r.historyProcessingDuration, _ = meter.Float64Histogram(MetricHistoryProcessingDuration, metric.WithUnit("s"))
	r.resourceAgeAtDeletion, _ = meter.Float64Histogram(MetricResourceAgeAtDeletion, metric.WithUnit("s"))

	r.activeResourcesCount, _ = meter.Int64UpDownCounter(MetricActiveResourcesCount, metric.WithUnit("1"))
	r.pendingDeletionsCount, _ = meter.Int64UpDownCounter(MetricPendingDeletionsCount, metric.WithUnit("1"))

	return r
}

// Timer represents a duration measurement that can be recorded when stopped
type Timer struct {
	start    time.Time
	recorder *Recorder
	labels   []attribute.KeyValue
}

// NewTimer creates a new timer for measuring durations
func (r *Recorder) NewTimer(labels ...attribute.KeyValue) *Timer {
	return &Timer{
		start:    time.Now(),
		recorder: r,
		labels:   labels,
	}
}

// RecordReconciliationDuration records the duration since the timer was created
func (t *Timer) RecordReconciliationDuration(ctx context.Context) {
	duration := time.Since(t.start).Seconds()
	t.recorder.reconciliationDuration.Record(ctx, duration, metric.WithAttributes(t.labels...))
}

// RecordTTLProcessingDuration records the duration since the timer was created
func (t *Timer) RecordTTLProcessingDuration(ctx context.Context) {
	duration := time.Since(t.start).Seconds()
	t.recorder.ttlProcessingDuration.Record(ctx, duration, metric.WithAttributes(t.labels...))
}

// RecordHistoryProcessingDuration records the duration since the timer was created
func (t *Timer) RecordHistoryProcessingDuration(ctx context.Context) {
	duration := time.Since(t.start).Seconds()
	t.recorder.historyProcessingDuration.Record(ctx, duration, metric.WithAttributes(t.labels...))
}

// RecordResourceProcessed increments the resources processed counter
// Note: This counts reconciliation events, not unique resources
func (r *Recorder) RecordResourceProcessed(ctx context.Context, resourceType, namespace, status string) {
	// Record in general metric
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
		attribute.String("status", status),
	}
	r.resourcesProcessed.Add(ctx, 1, metric.WithAttributes(labels...))

	// Record in specific resource type metrics
	specificLabels := []attribute.KeyValue{
		attribute.String("namespace", namespace),
		attribute.String("status", status),
	}

	switch resourceType {
	case ResourceTypePipelineRun:
		r.pipelineRunsProcessed.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	case ResourceTypeTaskRun:
		r.taskRunsProcessed.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	}
}

// RecordUniqueResourceProcessed increments counters only once per unique resource
func (r *Recorder) RecordUniqueResourceProcessed(ctx context.Context, resourceType, namespace, resourceName, status string) {
	// Use deletion tracker for deduplication (reuse the existing mechanism)
	tracker := GetDeletionTracker()
	if !tracker.RecordDeletion(ctx, resourceType, namespace, fmt.Sprintf("processed-%s", resourceName)) {
		// Already counted this resource processing, skip
		return
	}

	// Record in general metric
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
		attribute.String("status", status),
	}
	r.resourcesProcessed.Add(ctx, 1, metric.WithAttributes(labels...))

	// Record in specific resource type metrics
	specificLabels := []attribute.KeyValue{
		attribute.String("namespace", namespace),
		attribute.String("status", status),
	}

	switch resourceType {
	case ResourceTypePipelineRun:
		r.pipelineRunsProcessed.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	case ResourceTypeTaskRun:
		r.taskRunsProcessed.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	}
}

// RecordResourceDeleted increments the resources deleted counter and records age
// Uses deletion tracking to prevent double-counting when multiple workers
// process the same resource concurrently
func (r *Recorder) RecordResourceDeleted(ctx context.Context, resourceType, namespace, operation, resourceName string, resourceAge time.Duration) {
	// Check if this deletion should be counted (prevents double-counting)
	tracker := GetDeletionTracker()
	if !tracker.RecordDeletion(ctx, resourceType, namespace, resourceName) {
		return
	}

	// Record in general metrics
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
		attribute.String("operation", operation),
	}
	r.resourcesDeleted.Add(ctx, 1, metric.WithAttributes(labels...))
	r.resourceAgeAtDeletion.Record(ctx, resourceAge.Seconds(), metric.WithAttributes(labels...))

	// Record in specific resource type metrics
	specificLabels := []attribute.KeyValue{
		attribute.String("namespace", namespace),
		attribute.String("operation", operation),
	}

	switch resourceType {
	case ResourceTypePipelineRun:
		r.pipelineRunsDeleted.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	case ResourceTypeTaskRun:
		r.taskRunsDeleted.Add(ctx, 1, metric.WithAttributes(specificLabels...))
	}

	// Record deletion by reason
	deletionReason := DeletionReasonTTL
	if operation == OperationHistory {
		deletionReason = DeletionReasonLimits
	}

	reasonLabels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
		attribute.String("deletion_reason", deletionReason),
	}
	r.deletionsByReason.Add(ctx, 1, metric.WithAttributes(reasonLabels...))
}

// RecordResourceError increments the resources error counter
func (r *Recorder) RecordResourceError(ctx context.Context, resourceType, namespace, errorType, reason string) {
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
		attribute.String("error_type", errorType),
		attribute.String("reason", reason),
	}
	r.resourcesErrors.Add(ctx, 1, metric.WithAttributes(labels...))
}

// UpdateActiveResourcesCount updates the active resources gauge
func (r *Recorder) UpdateActiveResourcesCount(ctx context.Context, resourceType, namespace string, delta int64) {
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
	}
	r.activeResourcesCount.Add(ctx, delta, metric.WithAttributes(labels...))
}

// UpdatePendingDeletionsCount updates the pending deletions gauge
func (r *Recorder) UpdatePendingDeletionsCount(ctx context.Context, resourceType, namespace string, delta int64) {
	labels := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.String("namespace", namespace),
	}
	r.pendingDeletionsCount.Add(ctx, delta, metric.WithAttributes(labels...))
}

// ClassifyError determines the error type based on the error
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.IsNotFound(err):
		return "not_found"
	case errors.IsTimeout(err) || errors.IsServerTimeout(err):
		return "timeout"
	case errors.IsInvalid(err) || errors.IsBadRequest(err):
		return "validation"
	case errors.IsForbidden(err) || errors.IsUnauthorized(err):
		return "permission"
	case errors.IsInternalError(err) || errors.IsServiceUnavailable(err):
		return "api_error"
	default:
		return "internal"
	}
}
