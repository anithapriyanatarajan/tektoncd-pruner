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

package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = otel.GetMeterProvider().Meter(
		"github.com/anithapriyanatarajan/tektoncd-pruner",
		metric.WithInstrumentationVersion("v0.1.0"),
	)

	pipelineRunsPruned metric.Int64Counter
	taskRunsPruned     metric.Int64Counter
)

func init() {
	var err error
	pipelineRunsPruned, err = meter.Int64Counter(
		"tekton_pruner_pipelineruns_pruned_total",
		metric.WithDescription("Number of PipelineRuns pruned by namespace"),
		metric.WithUnit("{pipelineruns}"),
	)
	if err != nil {
		panic(err)
	}

	taskRunsPruned, err = meter.Int64Counter(
		"tekton_pruner_taskruns_pruned_total",
		metric.WithDescription("Number of TaskRuns pruned by namespace"),
		metric.WithUnit("{taskruns}"),
	)
	if err != nil {
		panic(err)
	}
}

// RecordPipelineRunPruned records when a PipelineRun is pruned
func RecordPipelineRunPruned(ctx context.Context, namespace string) {
	pipelineRunsPruned.Add(ctx, 1, metric.WithAttributes(
		attribute.String("namespace", namespace),
	))
}

// RecordTaskRunPruned records when a TaskRun is pruned
func RecordTaskRunPruned(ctx context.Context, namespace string) {
	taskRunsPruned.Add(ctx, 1, metric.WithAttributes(
		attribute.String("namespace", namespace),
	))
}
