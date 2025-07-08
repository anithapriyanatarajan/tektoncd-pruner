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

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sort"

	"github.com/openshift-pipelines/tektoncd-pruner/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

// HistoryLimiterResourceFuncs defines a set of methods that operate on resources
// with history limit capabilities.
type HistoryLimiterResourceFuncs interface {
	Type() string
	Get(ctx context.Context, namespace, name string) (metav1.Object, error)
	Update(ctx context.Context, resource metav1.Object) error
	Patch(ctx context.Context, namespace, name string, patchBytes []byte) error
	Delete(ctx context.Context, namespace, name string) error
	List(ctx context.Context, namespace, label string) ([]metav1.Object, error)
	GetFailedHistoryLimitCount(namespace, name string, selectors SelectorSpec) (*int32, string)
	GetSuccessHistoryLimitCount(namespace, name string, selectors SelectorSpec) (*int32, string)
	IsSuccessful(resource metav1.Object) bool
	IsFailed(resource metav1.Object) bool
	IsCompleted(resource metav1.Object) bool
	GetDefaultLabelKey() string
	GetEnforcedConfigLevel(namespace, name string, selectors SelectorSpec) EnforcedConfigLevel
}

// HistoryLimiter is a struct that encapsulates functionality for managing resources
// with history limits. It uses the HistoryLimiterResourceFuncs interface to interact
// with different types of resources
type HistoryLimiter struct {
	resourceFn HistoryLimiterResourceFuncs
}

// NewHistoryLimiter creates a new instance of HistoryLimiter, ensuring that the
// provided HistoryLimiterResourceFuncs interface is not nil
func NewHistoryLimiter(resourceFn HistoryLimiterResourceFuncs) (*HistoryLimiter, error) {
	hl := &HistoryLimiter{
		resourceFn: resourceFn,
	}
	if hl.resourceFn == nil {
		return nil, fmt.Errorf("resourceFunc interface can not be nil")
	}

	return hl, nil
}

// ProcessEvent processes the given resource for history limit cleanup
func (hl *HistoryLimiter) ProcessEvent(ctx context.Context, resource metav1.Object) error {
	logger := logging.FromContext(ctx)
	metrics := observability.GetGlobalMetrics()

	// Start history processing span
	ctx, span := observability.StartSpan(ctx, "history.process_event",
		attribute.String("resource.type", hl.resourceFn.Type()),
		attribute.String("resource.name", resource.GetName()),
		attribute.String("resource.namespace", resource.GetNamespace()),
	)
	defer span.End()

	labels := &observability.MetricLabels{
		Namespace:    resource.GetNamespace(),
		ResourceType: hl.resourceFn.Type(),
	}

	processStart := time.Now()
	defer func() {
		if metrics != nil {
			metrics.RecordHistoryProcessingDuration(ctx, labels, time.Since(processStart))
		}
	}()

	// Check if the resource is already processed
	if hl.isProcessed(resource) {
		observability.AddEvent(span, "resource.already_processed")
		labels.Reason = "already_processed"
		if metrics != nil {
			metrics.RecordResourceSkipped(ctx, labels, "already_processed")
		}
		observability.RecordSuccess(span)
		return nil
	}

	// Check if the resource is completed
	if !hl.resourceFn.IsCompleted(resource) {
		observability.AddEvent(span, "resource.not_completed")
		labels.Reason = "not_completed"
		if metrics != nil {
			metrics.RecordResourceSkipped(ctx, labels, "not_completed")
		}
		observability.RecordSuccess(span)
		return nil
	}

	// Mark the resource as processed
	hl.markAsProcessed(ctx, resource)

	observability.AddEvent(span, "resource.marked_as_processed")

	// Process cleanup for successful resources
	if hl.isSuccessfulResource(resource) {
		observability.AddEvent(span, "processing.successful_resource_cleanup")

		cleanupStart := time.Now()
		err := hl.DoSuccessfulResourceCleanup(ctx, resource)
		cleanupDuration := time.Since(cleanupStart)

		if err != nil {
			observability.RecordError(span, err, "successful_cleanup_error")
			observability.AddEvent(span, "cleanup.successful_failed",
				attribute.String("error", err.Error()),
				attribute.Float64("duration_seconds", cleanupDuration.Seconds()),
			)
			if metrics != nil {
				labels.Reason = "successful_cleanup_error"
				metrics.RecordResourceError(ctx, labels, "successful_cleanup")
			}
			return err
		}

		observability.AddEvent(span, "cleanup.successful_completed",
			attribute.Float64("duration_seconds", cleanupDuration.Seconds()),
		)

		if metrics != nil {
			metrics.RecordHistoryLimitEvent(ctx, labels)
		}
	}

	// Process cleanup for failed resources
	if hl.isFailedResource(resource) {
		observability.AddEvent(span, "processing.failed_resource_cleanup")

		cleanupStart := time.Now()
		err := hl.DoFailedResourceCleanup(ctx, resource)
		cleanupDuration := time.Since(cleanupStart)

		if err != nil {
			observability.RecordError(span, err, "failed_cleanup_error")
			observability.AddEvent(span, "cleanup.failed_failed",
				attribute.String("error", err.Error()),
				attribute.Float64("duration_seconds", cleanupDuration.Seconds()),
			)
			if metrics != nil {
				labels.Reason = "failed_cleanup_error"
				metrics.RecordResourceError(ctx, labels, "failed_cleanup")
			}
			return err
		}

		observability.AddEvent(span, "cleanup.failed_completed",
			attribute.Float64("duration_seconds", cleanupDuration.Seconds()),
		)

		if metrics != nil {
			metrics.RecordHistoryLimitEvent(ctx, labels)
		}
	}

	observability.RecordSuccess(span)
	logger.Debugw("History limit processing completed successfully",
		"resource", hl.resourceFn.Type(),
		"namespace", resource.GetNamespace(),
		"name", resource.GetName(),
		"successful", hl.isSuccessfulResource(resource),
		"failed", hl.isFailedResource(resource),
	)

	return nil
}

// adds an annotation, indicates this resource is already processed
// no action needed on the further reconcile loop for this Resource
// markAsProcessed patches the resource with the annotation 'mark as processed'
func (hl *HistoryLimiter) markAsProcessed(ctx context.Context, resource metav1.Object) {
	logger := logging.FromContext(ctx)

	logger.Debugw("marking resource as processed", "resource", hl.resourceFn.Type(), "namespace", resource.GetNamespace(), "name", resource.GetName())

	// Fetch the latest version of the resource
	resourceLatest, err := hl.resourceFn.Get(ctx, resource.GetNamespace(), resource.GetName())
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		logger.Errorw("error getting resource", "resource", hl.resourceFn.Type(),
			"namespace", resource.GetNamespace(), "name", resource.GetName(), zap.Error(err))
		return
	}

	// Prepare the annotation update
	processedTimeAsString := time.Now().Format(time.RFC3339)
	annotations := resourceLatest.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[AnnotationHistoryLimitCheckProcessed] = processedTimeAsString

	// Create a patch with the new annotations
	patchData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	}

	// Convert patchData to JSON
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		logger.Errorw("error marshaling patch data", zap.Error(err))
		return
	}

	// Apply the patch
	err = hl.resourceFn.Patch(ctx, resourceLatest.GetNamespace(), resourceLatest.GetName(), patchBytes)
	if err != nil {
		logger.Errorw("error patching resource with 'mark as processed' annotation",
			"resource", hl.resourceFn.Type(), "namespace", resourceLatest.GetNamespace(), "name", resourceLatest.GetName(), zap.Error(err))
	}
}

func (hl *HistoryLimiter) isProcessed(resource metav1.Object) bool {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, found := annotations[AnnotationHistoryLimitCheckProcessed]
	return found
}

func (hl *HistoryLimiter) DoSuccessfulResourceCleanup(ctx context.Context, resource metav1.Object) error {
	logging := logging.FromContext(ctx)

	logging.Debugw("processing a successful resource", "resource", hl.resourceFn.Type(), "namespace", resource.GetNamespace(), "name", resource.GetName())
	return hl.doResourceCleanup(ctx, resource, AnnotationSuccessfulHistoryLimit, hl.resourceFn.GetSuccessHistoryLimitCount, hl.isSuccessfulResource)
}

func (hl *HistoryLimiter) DoFailedResourceCleanup(ctx context.Context, resource metav1.Object) error {
	logging := logging.FromContext(ctx)
	logging.Debugw("processing a failed resource", "resource", hl.resourceFn.Type(), "namespace", resource.GetNamespace(), "name", resource.GetName())
	return hl.doResourceCleanup(ctx, resource, AnnotationFailedHistoryLimit, hl.resourceFn.GetFailedHistoryLimitCount, hl.isFailedResource)
}

// isFailedResource checks if a resource has failed
func (hl *HistoryLimiter) isFailedResource(resource metav1.Object) bool {
	return hl.resourceFn.IsFailed(resource)
}

// isSuccessfulResource checks if a resource is successful
func (hl *HistoryLimiter) isSuccessfulResource(resource metav1.Object) bool {
	return hl.resourceFn.IsSuccessful(resource)
}

// doResourceCleanup handles cleanup for a resource based on the provided filter function
func (hl *HistoryLimiter) doResourceCleanup(ctx context.Context, resource metav1.Object, historyLimitAnnotation string, getHistoryLimitFn func(string, string, SelectorSpec) (*int32, string), getResourceFilterFn func(metav1.Object) bool) error {
	metrics := observability.GetGlobalMetrics()

	// Start cleanup span
	ctx, span := observability.StartSpan(ctx, "history.cleanup",
		attribute.String("resource.type", hl.resourceFn.Type()),
		attribute.String("resource.name", resource.GetName()),
		attribute.String("resource.namespace", resource.GetNamespace()),
		attribute.String("cleanup.type", historyLimitAnnotation),
	)
	defer span.End()

	labels := &observability.MetricLabels{
		Namespace:    resource.GetNamespace(),
		ResourceType: hl.resourceFn.Type(),
		Reason:       historyLimitAnnotation,
	}

	logger := logging.FromContext(ctx)

	// Obtain the resource name and selectors first
	labelKey := getResourceNameLabelKey(resource, hl.resourceFn.GetDefaultLabelKey())
	resourceName := getResourceName(resource, labelKey)
	resourceSelectors := hl.getResourceSelectors(resource)

	// Check the enforced configuration level
	enforcedLevel := hl.resourceFn.GetEnforcedConfigLevel(resource.GetNamespace(), resourceName, resourceSelectors)

	observability.AddEvent(span, "cleanup.config_check",
		attribute.String("config.enforced_level", string(enforcedLevel)),
		attribute.String("resource.name", resourceName),
	)

	// Get the history limit configuration
	historyLimit, identifiedBy := getHistoryLimitFn(resource.GetNamespace(), resourceName, resourceSelectors)
	if historyLimit == nil {
		observability.AddEvent(span, "cleanup.no_limit_configured")
		labels.Reason = "no_limit_configured"
		if metrics != nil {
			metrics.RecordResourceSkipped(ctx, labels, "no_limit_configured")
		}
		observability.RecordSuccess(span)
		return nil
	}

	observability.AddEvent(span, "cleanup.limit_found",
		attribute.Int("history.limit", int(*historyLimit)),
		attribute.String("config.source", identifiedBy),
	)

	// Fetch all resources matching the criteria
	listStart := time.Now()
	resources, err := hl.resourceFn.List(ctx, resource.GetNamespace(), fmt.Sprintf("%s=%s", labelKey, resourceName))
	listDuration := time.Since(listStart)

	if err != nil {
		observability.RecordError(span, err, "resource_list_error")
		observability.AddEvent(span, "cleanup.list_failed",
			attribute.String("error", err.Error()),
			attribute.Float64("duration_seconds", listDuration.Seconds()),
		)
		return err
	}

	observability.AddEvent(span, "cleanup.resources_listed",
		attribute.Int("resource.count", len(resources)),
		attribute.Float64("duration_seconds", listDuration.Seconds()),
	)

	// Filter resources by completion status and type (successful/failed)
	var completedResources []metav1.Object
	for _, res := range resources {
		if hl.resourceFn.IsCompleted(res) && getResourceFilterFn(res) {
			completedResources = append(completedResources, res)
		}
	}

	observability.AddEvent(span, "cleanup.resources_filtered",
		attribute.Int("completed.count", len(completedResources)),
	)

	// Check if cleanup is needed
	if len(completedResources) <= int(*historyLimit) {
		observability.AddEvent(span, "cleanup.within_limit",
			attribute.Int("current.count", len(completedResources)),
			attribute.Int("limit", int(*historyLimit)),
		)
		labels.Reason = "within_limit"
		if metrics != nil {
			metrics.RecordResourceSkipped(ctx, labels, "within_limit")
		}
		observability.RecordSuccess(span)
		return nil
	}

	// Sort resources by creation timestamp (oldest first)
	sort.Slice(completedResources, func(i, j int) bool {
		return completedResources[i].GetCreationTimestamp().Time.Before(completedResources[j].GetCreationTimestamp().Time)
	})

	// Calculate how many resources to delete
	resourcesToDelete := len(completedResources) - int(*historyLimit)

	observability.AddEvent(span, "cleanup.deletion_required",
		attribute.Int("resources.to_delete", resourcesToDelete),
		attribute.Int("current.count", len(completedResources)),
		attribute.Int("limit", int(*historyLimit)),
	)

	// Delete excess resources
	deletedCount := 0
	for i := 0; i < resourcesToDelete; i++ {
		res := completedResources[i]

		deleteStart := time.Now()
		err := hl.resourceFn.Delete(ctx, res.GetNamespace(), res.GetName())
		deleteDuration := time.Since(deleteStart)

		if err != nil {
			if errors.IsNotFound(err) {
				// Resource already deleted, continue
				observability.AddEvent(span, "cleanup.resource_already_deleted",
					attribute.String("resource.name", res.GetName()),
				)
				continue
			}

			observability.RecordError(span, err, "resource_delete_error")
			observability.AddEvent(span, "cleanup.delete_failed",
				attribute.String("resource.name", res.GetName()),
				attribute.String("error", err.Error()),
				attribute.Float64("duration_seconds", deleteDuration.Seconds()),
			)

			if metrics != nil {
				resourceLabels := &observability.MetricLabels{
					Namespace:    res.GetNamespace(),
					ResourceType: hl.resourceFn.Type(),
					Reason:       "deletion_error",
				}
				metrics.RecordResourceDeleteError(ctx, resourceLabels, "history_cleanup")
			}

			return fmt.Errorf("failed to delete resource %s/%s: %w", res.GetNamespace(), res.GetName(), err)
		}

		deletedCount++

		// Calculate resource age for metrics
		var resourceAge float64
		if !res.GetCreationTimestamp().Time.IsZero() {
			resourceAge = time.Since(res.GetCreationTimestamp().Time).Seconds()
		}

		observability.AddEvent(span, "cleanup.resource_deleted",
			attribute.String("resource.name", res.GetName()),
			attribute.Float64("resource_age_seconds", resourceAge),
			attribute.Float64("duration_seconds", deleteDuration.Seconds()),
		)

		if metrics != nil {
			resourceLabels := &observability.MetricLabels{
				Namespace:    res.GetNamespace(),
				ResourceType: hl.resourceFn.Type(),
				Reason:       "history_limit",
				Status:       "deleted",
			}
			metrics.RecordResourceDeleted(ctx, resourceLabels, resourceAge)
			metrics.RecordResourceCleanedByHistory(ctx, resourceLabels)
			metrics.RecordResourceDeletionDuration(ctx, resourceLabels, deleteDuration)
		}

		logger.Debugw("Resource deleted due to history limit",
			"resource", hl.resourceFn.Type(),
			"namespace", res.GetNamespace(),
			"name", res.GetName(),
			"age", resourceAge,
			"historyLimit", *historyLimit,
		)
	}

	observability.AddEvent(span, "cleanup.completed",
		attribute.Int("resources.deleted", deletedCount),
		attribute.Int("resources.remaining", len(completedResources)-deletedCount),
	)

	observability.RecordSuccess(span)

	logger.Infow("History-based cleanup completed",
		"resource", hl.resourceFn.Type(),
		"namespace", resource.GetNamespace(),
		"historyLimit", *historyLimit,
		"totalCompleted", len(completedResources),
		"deleted", deletedCount,
		"remaining", len(completedResources)-deletedCount,
	)

	return nil
}

// getResourceSelectors constructs the selector spec for a resource
func (hl *HistoryLimiter) getResourceSelectors(resource metav1.Object) SelectorSpec {
	selectors := SelectorSpec{}
	if annotations := resource.GetAnnotations(); len(annotations) > 0 {
		selectors.MatchAnnotations = annotations
	}
	if labels := resource.GetLabels(); len(labels) > 0 {
		selectors.MatchLabels = labels
	}
	return selectors
}
