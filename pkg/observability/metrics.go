/*
Copyright 2024 The Tekton Authors
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

package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

// MetricLabels defines standard labels for metrics
type MetricLabels struct {
	Namespace    string
	ResourceType string
	Reason       string
	Status       string
	ConfigLevel  string
}

// ToAttributes converts MetricLabels to OpenTelemetry attributes
func (ml *MetricLabels) ToAttributes() []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	if ml.Namespace != "" {
		attrs = append(attrs, attribute.String("namespace", ml.Namespace))
	}
	if ml.ResourceType != "" {
		attrs = append(attrs, attribute.String("resource_type", ml.ResourceType))
	}
	if ml.Reason != "" {
		attrs = append(attrs, attribute.String("reason", ml.Reason))
	}
	if ml.Status != "" {
		attrs = append(attrs, attribute.String("status", ml.Status))
	}
	if ml.ConfigLevel != "" {
		attrs = append(attrs, attribute.String("config_level", ml.ConfigLevel))
	}

	return attrs
}

// PrunerMetrics holds all the metrics for the tektoncd-pruner
type PrunerMetrics struct {
	// Resource processing metrics
	resourcesProcessedTotal metric.Int64Counter
	resourcesDeletedTotal   metric.Int64Counter
	resourcesErrorsTotal    metric.Int64Counter
	resourcesSkippedTotal   metric.Int64Counter

	// Performance metrics
	reconciliationDuration    metric.Float64Histogram
	ttlProcessingDuration     metric.Float64Histogram
	historyProcessingDuration metric.Float64Histogram
	resourceDeletionDuration  metric.Float64Histogram

	// Queue and backlog metrics
	resourcesQueuedTotal   metric.Int64Counter
	currentResourcesQueued metric.Int64UpDownCounter

	// TTL-specific metrics
	ttlAnnotationUpdatesTotal metric.Int64Counter
	ttlExpirationEventsTotal  metric.Int64Counter

	// History limit metrics
	historyLimitEventsTotal   metric.Int64Counter
	resourcesCleanedByHistory metric.Int64Counter

	// Configuration metrics
	configurationReloadsTotal metric.Int64Counter
	configurationErrorsTotal  metric.Int64Counter

	// Resource age metrics
	resourceAgeAtDeletion metric.Float64Histogram

	// Error breakdown metrics
	resourceDeleteErrorsTotal metric.Int64Counter
	resourceUpdateErrorsTotal metric.Int64Counter

	// Gauge metrics for current state
	activeResourcesCount metric.Int64UpDownCounter

	// Internal
	meter  metric.Meter
	logger *zap.SugaredLogger
	once   sync.Once
}

// NewPrunerMetrics creates a new instance of PrunerMetrics
func NewPrunerMetrics(ctx context.Context, meterProvider metric.MeterProvider) (*PrunerMetrics, error) {
	logger := logging.FromContext(ctx)

	meter := meterProvider.Meter(
		"github.com/openshift-pipelines/tektoncd-pruner",
		metric.WithInstrumentationVersion("v1.0.0"),
		metric.WithSchemaURL("https://opentelemetry.io/schemas/1.24.0"),
	)

	pm := &PrunerMetrics{
		meter:  meter,
		logger: logger,
	}

	if err := pm.initializeMetrics(); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	logger.Info("Pruner metrics initialized successfully")
	return pm, nil
}

// initializeMetrics creates all the metric instruments
func (pm *PrunerMetrics) initializeMetrics() error {
	var err error

	// Resource processing counters
	pm.resourcesProcessedTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_processed_total",
		metric.WithDescription("Total number of resources processed by the pruner"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_processed_total counter: %w", err)
	}

	pm.resourcesDeletedTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_deleted_total",
		metric.WithDescription("Total number of resources deleted by the pruner"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_deleted_total counter: %w", err)
	}

	pm.resourcesErrorsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_errors_total",
		metric.WithDescription("Total number of errors encountered while processing resources"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_errors_total counter: %w", err)
	}

	pm.resourcesSkippedTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_skipped_total",
		metric.WithDescription("Total number of resources skipped by the pruner"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_skipped_total counter: %w", err)
	}

	// Performance histograms
	pm.reconciliationDuration, err = pm.meter.Float64Histogram(
		"tektoncd_pruner_reconciliation_duration_seconds",
		metric.WithDescription("Time spent in reconciliation"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create reconciliation_duration histogram: %w", err)
	}

	pm.ttlProcessingDuration, err = pm.meter.Float64Histogram(
		"tektoncd_pruner_ttl_processing_duration_seconds",
		metric.WithDescription("Time spent processing TTL for resources"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create ttl_processing_duration histogram: %w", err)
	}

	pm.historyProcessingDuration, err = pm.meter.Float64Histogram(
		"tektoncd_pruner_history_processing_duration_seconds",
		metric.WithDescription("Time spent processing history limits for resources"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create history_processing_duration histogram: %w", err)
	}

	pm.resourceDeletionDuration, err = pm.meter.Float64Histogram(
		"tektoncd_pruner_resource_deletion_duration_seconds",
		metric.WithDescription("Time taken to delete individual resources"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_deletion_duration histogram: %w", err)
	}

	// Queue metrics
	pm.resourcesQueuedTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_queued_total",
		metric.WithDescription("Total number of resources queued for processing"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_queued_total counter: %w", err)
	}

	pm.currentResourcesQueued, err = pm.meter.Int64UpDownCounter(
		"tektoncd_pruner_current_resources_queued",
		metric.WithDescription("Current number of resources in processing queue"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create current_resources_queued gauge: %w", err)
	}

	// TTL-specific metrics
	pm.ttlAnnotationUpdatesTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_ttl_annotation_updates_total",
		metric.WithDescription("Total number of TTL annotation updates"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create ttl_annotation_updates_total counter: %w", err)
	}

	pm.ttlExpirationEventsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_ttl_expiration_events_total",
		metric.WithDescription("Total number of TTL expiration events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create ttl_expiration_events_total counter: %w", err)
	}

	// History limit metrics
	pm.historyLimitEventsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_history_limit_events_total",
		metric.WithDescription("Total number of history limit events triggered"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create history_limit_events_total counter: %w", err)
	}

	pm.resourcesCleanedByHistory, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resources_cleaned_by_history",
		metric.WithDescription("Total number of resources cleaned due to history limits"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resources_cleaned_by_history counter: %w", err)
	}

	// Configuration metrics
	pm.configurationReloadsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_configuration_reloads_total",
		metric.WithDescription("Total number of configuration reloads"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create configuration_reloads_total counter: %w", err)
	}

	pm.configurationErrorsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_configuration_errors_total",
		metric.WithDescription("Total number of configuration errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create configuration_errors_total counter: %w", err)
	}

	// Resource age metrics
	pm.resourceAgeAtDeletion, err = pm.meter.Float64Histogram(
		"tektoncd_pruner_resource_age_at_deletion_seconds",
		metric.WithDescription("Age of resources when they were deleted"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			60,      // 1 minute
			300,     // 5 minutes
			1800,    // 30 minutes
			3600,    // 1 hour
			7200,    // 2 hours
			21600,   // 6 hours
			86400,   // 1 day
			604800,  // 1 week
			2592000, // 1 month
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_age_at_deletion histogram: %w", err)
	}

	// Error breakdown metrics
	pm.resourceDeleteErrorsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resource_delete_errors_total",
		metric.WithDescription("Total number of resource deletion errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_delete_errors_total counter: %w", err)
	}

	pm.resourceUpdateErrorsTotal, err = pm.meter.Int64Counter(
		"tektoncd_pruner_resource_update_errors_total",
		metric.WithDescription("Total number of resource update errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_update_errors_total counter: %w", err)
	}

	// Gauge for active resources
	pm.activeResourcesCount, err = pm.meter.Int64UpDownCounter(
		"tektoncd_pruner_active_resources",
		metric.WithDescription("Current number of active resources being tracked"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return fmt.Errorf("failed to create active_resources gauge: %w", err)
	}

	return nil
}

// Metric recording methods

// RecordResourceProcessed records that a resource has been processed
func (pm *PrunerMetrics) RecordResourceProcessed(ctx context.Context, labels *MetricLabels) {
	pm.resourcesProcessedTotal.Add(ctx, 1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordResourceDeleted records that a resource has been deleted
func (pm *PrunerMetrics) RecordResourceDeleted(ctx context.Context, labels *MetricLabels, ageSeconds float64) {
	attrs := labels.ToAttributes()
	pm.resourcesDeletedTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	pm.resourceAgeAtDeletion.Record(ctx, ageSeconds, metric.WithAttributes(attrs...))
}

// RecordResourceError records an error processing a resource
func (pm *PrunerMetrics) RecordResourceError(ctx context.Context, labels *MetricLabels, errorType string) {
	attrs := append(labels.ToAttributes(), attribute.String("error_type", errorType))
	pm.resourcesErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordResourceSkipped records that a resource was skipped
func (pm *PrunerMetrics) RecordResourceSkipped(ctx context.Context, labels *MetricLabels, reason string) {
	attrs := append(labels.ToAttributes(), attribute.String("skip_reason", reason))
	pm.resourcesSkippedTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordReconciliationDuration records the time spent in reconciliation
func (pm *PrunerMetrics) RecordReconciliationDuration(ctx context.Context, labels *MetricLabels, duration time.Duration) {
	pm.reconciliationDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(labels.ToAttributes()...))
}

// RecordTTLProcessingDuration records the time spent processing TTL
func (pm *PrunerMetrics) RecordTTLProcessingDuration(ctx context.Context, labels *MetricLabels, duration time.Duration) {
	pm.ttlProcessingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(labels.ToAttributes()...))
}

// RecordHistoryProcessingDuration records the time spent processing history limits
func (pm *PrunerMetrics) RecordHistoryProcessingDuration(ctx context.Context, labels *MetricLabels, duration time.Duration) {
	pm.historyProcessingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(labels.ToAttributes()...))
}

// RecordResourceDeletionDuration records the time spent deleting a resource
func (pm *PrunerMetrics) RecordResourceDeletionDuration(ctx context.Context, labels *MetricLabels, duration time.Duration) {
	pm.resourceDeletionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(labels.ToAttributes()...))
}

// RecordResourceQueued records that a resource has been queued
func (pm *PrunerMetrics) RecordResourceQueued(ctx context.Context, labels *MetricLabels) {
	attrs := labels.ToAttributes()
	pm.resourcesQueuedTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	pm.currentResourcesQueued.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordResourceDequeued records that a resource has been dequeued
func (pm *PrunerMetrics) RecordResourceDequeued(ctx context.Context, labels *MetricLabels) {
	pm.currentResourcesQueued.Add(ctx, -1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordTTLAnnotationUpdate records a TTL annotation update
func (pm *PrunerMetrics) RecordTTLAnnotationUpdate(ctx context.Context, labels *MetricLabels) {
	pm.ttlAnnotationUpdatesTotal.Add(ctx, 1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordTTLExpiration records a TTL expiration event
func (pm *PrunerMetrics) RecordTTLExpiration(ctx context.Context, labels *MetricLabels) {
	pm.ttlExpirationEventsTotal.Add(ctx, 1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordHistoryLimitEvent records a history limit event
func (pm *PrunerMetrics) RecordHistoryLimitEvent(ctx context.Context, labels *MetricLabels) {
	pm.historyLimitEventsTotal.Add(ctx, 1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordResourceCleanedByHistory records that a resource was cleaned by history limits
func (pm *PrunerMetrics) RecordResourceCleanedByHistory(ctx context.Context, labels *MetricLabels) {
	pm.resourcesCleanedByHistory.Add(ctx, 1, metric.WithAttributes(labels.ToAttributes()...))
}

// RecordConfigurationReload records a configuration reload
func (pm *PrunerMetrics) RecordConfigurationReload(ctx context.Context) {
	pm.configurationReloadsTotal.Add(ctx, 1)
}

// RecordConfigurationError records a configuration error
func (pm *PrunerMetrics) RecordConfigurationError(ctx context.Context, errorType string) {
	pm.configurationErrorsTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("error_type", errorType)))
}

// RecordResourceDeleteError records a resource deletion error
func (pm *PrunerMetrics) RecordResourceDeleteError(ctx context.Context, labels *MetricLabels, errorType string) {
	attrs := append(labels.ToAttributes(), attribute.String("error_type", errorType))
	pm.resourceDeleteErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordResourceUpdateError records a resource update error
func (pm *PrunerMetrics) RecordResourceUpdateError(ctx context.Context, labels *MetricLabels, errorType string) {
	attrs := append(labels.ToAttributes(), attribute.String("error_type", errorType))
	pm.resourceUpdateErrorsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// UpdateActiveResourcesCount updates the active resources count
func (pm *PrunerMetrics) UpdateActiveResourcesCount(ctx context.Context, labels *MetricLabels, count int64) {
	pm.activeResourcesCount.Add(ctx, count, metric.WithAttributes(labels.ToAttributes()...))
}

// GetMeter returns the underlying OpenTelemetry meter for custom metrics
func (pm *PrunerMetrics) GetMeter() metric.Meter {
	return pm.meter
}

// Global metrics instance (singleton pattern)
var (
	globalMetrics *PrunerMetrics
	metricsOnce   sync.Once
)

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *PrunerMetrics {
	return globalMetrics
}

// InitializeGlobalMetrics initializes the global metrics instance
func InitializeGlobalMetrics(ctx context.Context, meterProvider metric.MeterProvider) error {
	var err error
	metricsOnce.Do(func() {
		globalMetrics, err = NewPrunerMetrics(ctx, meterProvider)
	})
	return err
}

// MustGetGlobalMetrics returns the global metrics instance or panics if not initialized
func MustGetGlobalMetrics() *PrunerMetrics {
	if globalMetrics == nil {
		panic("global metrics not initialized - call InitializeGlobalMetrics first")
	}
	return globalMetrics
}
