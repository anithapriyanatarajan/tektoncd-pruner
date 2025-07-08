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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

const (
	// Tracer name for all pruner traces
	TracerName = "github.com/openshift-pipelines/tektoncd-pruner"

	// Common span names
	SpanReconciliation    = "reconciliation"
	SpanTTLProcessing     = "ttl_processing"
	SpanHistoryProcessing = "history_processing"
	SpanResourceDeletion  = "resource_deletion"
	SpanResourceUpdate    = "resource_update"
	SpanResourceList      = "resource_list"
	SpanConfigurationLoad = "configuration_load"
	SpanGarbageCollection = "garbage_collection"

	// Attribute keys
	AttrResourceType      = "resource.type"
	AttrResourceName      = "resource.name"
	AttrResourceNamespace = "resource.namespace"
	AttrResourceUID       = "resource.uid"
	AttrOperation         = "operation"
	AttrReason            = "reason"
	AttrConfigLevel       = "config.level"
	AttrTTLSeconds        = "ttl.seconds"
	AttrHistoryLimit      = "history.limit"
	AttrErrorType         = "error.type"
	AttrResourceCount     = "resource.count"
)

// TracingHelper provides convenient methods for creating and managing traces
type TracingHelper struct {
	tracer trace.Tracer
	logger *zap.SugaredLogger
}

// NewTracingHelper creates a new tracing helper
func NewTracingHelper(ctx context.Context, tracerProvider trace.TracerProvider) *TracingHelper {
	logger := logging.FromContext(ctx)

	tracer := tracerProvider.Tracer(
		TracerName,
		trace.WithInstrumentationVersion("v1.0.0"),
		trace.WithSchemaURL("https://opentelemetry.io/schemas/1.24.0"),
	)

	return &TracingHelper{
		tracer: tracer,
		logger: logger,
	}
}

// StartSpan starts a new span with the given name and attributes
func (th *TracingHelper) StartSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := th.tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
	return ctx, span
}

// StartReconciliationSpan starts a span for resource reconciliation
func (th *TracingHelper) StartReconciliationSpan(ctx context.Context, resourceType string, resource metav1.Object) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrResourceUID, string(resource.GetUID())),
		attribute.String(AttrOperation, "reconcile"),
	}

	spanName := fmt.Sprintf("%s.%s", SpanReconciliation, resourceType)
	return th.StartSpan(ctx, spanName, attrs...)
}

// StartTTLProcessingSpan starts a span for TTL processing
func (th *TracingHelper) StartTTLProcessingSpan(ctx context.Context, resourceType string, resource metav1.Object, ttlSeconds *int32) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrOperation, "ttl_processing"),
	}

	if ttlSeconds != nil {
		attrs = append(attrs, attribute.Int(AttrTTLSeconds, int(*ttlSeconds)))
	}

	return th.StartSpan(ctx, SpanTTLProcessing, attrs...)
}

// StartHistoryProcessingSpan starts a span for history limit processing
func (th *TracingHelper) StartHistoryProcessingSpan(ctx context.Context, resourceType string, resource metav1.Object, historyLimit *int32) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrOperation, "history_processing"),
	}

	if historyLimit != nil {
		attrs = append(attrs, attribute.Int(AttrHistoryLimit, int(*historyLimit)))
	}

	return th.StartSpan(ctx, SpanHistoryProcessing, attrs...)
}

// StartResourceDeletionSpan starts a span for resource deletion
func (th *TracingHelper) StartResourceDeletionSpan(ctx context.Context, resourceType string, resource metav1.Object, reason string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrOperation, "delete"),
		attribute.String(AttrReason, reason),
	}

	return th.StartSpan(ctx, SpanResourceDeletion, attrs...)
}

// StartResourceUpdateSpan starts a span for resource update
func (th *TracingHelper) StartResourceUpdateSpan(ctx context.Context, resourceType string, resource metav1.Object, operation string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrOperation, operation),
	}

	return th.StartSpan(ctx, SpanResourceUpdate, attrs...)
}

// StartResourceListSpan starts a span for resource listing operations
func (th *TracingHelper) StartResourceListSpan(ctx context.Context, resourceType, namespace string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrOperation, "list"),
	}

	if namespace != "" {
		attrs = append(attrs, attribute.String(AttrResourceNamespace, namespace))
	}

	return th.StartSpan(ctx, SpanResourceList, attrs...)
}

// StartConfigurationLoadSpan starts a span for configuration loading
func (th *TracingHelper) StartConfigurationLoadSpan(ctx context.Context) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrOperation, "configuration_load"),
	}

	return th.StartSpan(ctx, SpanConfigurationLoad, attrs...)
}

// StartGarbageCollectionSpan starts a span for garbage collection operations
func (th *TracingHelper) StartGarbageCollectionSpan(ctx context.Context, namespaces []string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String(AttrOperation, "garbage_collection"),
		attribute.Int(AttrResourceCount, len(namespaces)),
	}

	return th.StartSpan(ctx, SpanGarbageCollection, attrs...)
}

// RecordError records an error in the current span
func (th *TracingHelper) RecordError(span trace.Span, err error, errorType string) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(attribute.String(AttrErrorType, errorType))
}

// RecordResourceAttributes adds resource-specific attributes to a span
func (th *TracingHelper) RecordResourceAttributes(span trace.Span, resourceType string, resource metav1.Object) {
	if resource == nil {
		return
	}

	span.SetAttributes(
		attribute.String(AttrResourceType, resourceType),
		attribute.String(AttrResourceName, resource.GetName()),
		attribute.String(AttrResourceNamespace, resource.GetNamespace()),
		attribute.String(AttrResourceUID, string(resource.GetUID())),
	)
}

// RecordSuccess marks a span as successful
func (th *TracingHelper) RecordSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}

// AddEvent adds an event to the current span with attributes
func (th *TracingHelper) AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// WithSpanContext wraps a function with span context
func (th *TracingHelper) WithSpanContext(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := th.StartSpan(ctx, spanName, attrs...)
	defer span.End()

	if err := fn(ctx); err != nil {
		th.RecordError(span, err, "execution_error")
		return err
	}

	th.RecordSuccess(span)
	return nil
}

// GetTracer returns the underlying OpenTelemetry tracer
func (th *TracingHelper) GetTracer() trace.Tracer {
	return th.tracer
}

// SpanFromContext returns the current span from the context
func (th *TracingHelper) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a context with the given span
func (th *TracingHelper) ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// Global tracing helper instance
var (
	globalTracingHelper *TracingHelper
	tracingOnce         sync.Once
)

// GetGlobalTracingHelper returns the global tracing helper instance
func GetGlobalTracingHelper() *TracingHelper {
	return globalTracingHelper
}

// InitializeGlobalTracingHelper initializes the global tracing helper instance
func InitializeGlobalTracingHelper(ctx context.Context, tracerProvider trace.TracerProvider) {
	tracingOnce.Do(func() {
		globalTracingHelper = NewTracingHelper(ctx, tracerProvider)
	})
}

// MustGetGlobalTracingHelper returns the global tracing helper or panics if not initialized
func MustGetGlobalTracingHelper() *TracingHelper {
	if globalTracingHelper == nil {
		panic("global tracing helper not initialized - call InitializeGlobalTracingHelper first")
	}
	return globalTracingHelper
}

// Convenience functions for common operations

// StartSpan is a convenience function that uses the global tracing helper
func StartSpan(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if th := GetGlobalTracingHelper(); th != nil {
		return th.StartSpan(ctx, spanName, attrs...)
	}
	// Return a no-op span if tracing is not initialized
	return ctx, trace.SpanFromContext(ctx)
}

// RecordError is a convenience function that uses the global tracing helper
func RecordError(span trace.Span, err error, errorType string) {
	if th := GetGlobalTracingHelper(); th != nil {
		th.RecordError(span, err, errorType)
	}
}

// RecordSuccess is a convenience function that uses the global tracing helper
func RecordSuccess(span trace.Span) {
	if th := GetGlobalTracingHelper(); th != nil {
		th.RecordSuccess(span)
	}
}

// AddEvent is a convenience function that uses the global tracing helper
func AddEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if th := GetGlobalTracingHelper(); th != nil {
		th.AddEvent(span, name, attrs...)
	}
}

// WithSpan wraps a function with a span using the global tracing helper
func WithSpan(ctx context.Context, spanName string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	if th := GetGlobalTracingHelper(); th != nil {
		return th.WithSpanContext(ctx, spanName, attrs, fn)
	}
	// If tracing is not initialized, just execute the function
	return fn(ctx)
}
