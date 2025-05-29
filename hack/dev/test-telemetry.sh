#!/bin/bash

# Exit on any error
set -e

echo "Creating Kind cluster..."
kind create cluster --name tekton-pruner-dev

echo "Creating tekton-pipelines namespace..."
kubectl create namespace tekton-pipelines

echo "Deploying OpenTelemetry Collector..."
kubectl apply -f config/otel-collector-config.yaml

echo "Waiting for OpenTelemetry Collector to be ready..."
kubectl wait --for=condition=Available deployment/otel-collector -n tekton-pipelines --timeout=60s

echo "Deploying Tekton Pruner..."
# Install pruner's RBAC resources
kubectl apply -f config/200-serviceaccount.yaml
kubectl apply -f config/200-role.yaml
kubectl apply -f config/200-clusterrole.yaml
kubectl apply -f config/201-rolebinding.yaml
kubectl apply -f config/201-clusterrolebinding.yaml

# Install pruner's config
kubectl apply -f config/600-tekton-pruner-default-spec.yaml
kubectl apply -f config/config-logging.yaml
kubectl apply -f config/config-observability.yaml

# Install pruner controller
kubectl apply -f config/controller.yaml

echo "Waiting for Tekton Pruner to be ready..."
kubectl wait --for=condition=Available deployment/tekton-pruner-controller -n tekton-pipelines --timeout=60s

echo "Setting up port-forward for Prometheus metrics..."
kubectl port-forward service/otel-collector -n tekton-pipelines 8889:8889 &
PF_PID=$!

echo "Waiting for port-forward to be ready..."
sleep 5

echo "You can now access the metrics at: http://localhost:8889/metrics"
echo "Sample metrics to look for:"
echo "  - tekton_pruner_pipelineruns_pruned_total"
echo "  - tekton_pruner_taskruns_pruned_total"
echo ""
echo "To create test PipelineRuns and TaskRuns, run:"
echo "kubectl create -f test/samples/"
echo ""
echo "Press Ctrl+C to stop port-forwarding and clean up"

# Wait for Ctrl+C
trap "kill $PF_PID" INT
wait
