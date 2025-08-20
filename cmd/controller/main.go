package main

import (
	"flag"
	"os"
	"strings"

	"github.com/openshift-pipelines/tektoncd-pruner/pkg/metrics"
	"github.com/openshift-pipelines/tektoncd-pruner/pkg/reconciler/pipelinerun"
	"github.com/openshift-pipelines/tektoncd-pruner/pkg/reconciler/taskrun"
	"github.com/openshift-pipelines/tektoncd-pruner/pkg/reconciler/tektonpruner"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

func main() {
	// Set metrics domain for proper prefixing of Knative controller metrics
	os.Setenv("METRICS_DOMAIN", "tekton-pruner-controller")

	// Command-line flags
	flag.IntVar(&controller.DefaultThreadsPerController, "threads-per-controller", controller.DefaultThreadsPerController, "Threads per controller")
	namespace := flag.String("namespace", corev1.NamespaceAll, "Namespace to watch. Defaults to all namespaces.")
	disableHA := flag.Bool("disable-ha", true, "Disable high-availability")
	flag.Parse()

	// Setup context and logging
	ctx := signals.NewContext()
	logger := logging.FromContext(ctx)

	// Initialize pruner-specific metrics (simple setup)
	if err := metrics.GetExporter().Initialize(ctx, metrics.DefaultMetricsConfig()); err != nil {
		logger.Errorf("Failed to initialize pruner metrics: %v", err)
	}

	// REST config
	cfg := injection.ParseAndGetRESTConfigOrDie()
	cfg.QPS = 2 * rest.DefaultQPS
	cfg.Burst = 2 * rest.DefaultBurst

	// Handle namespace scoping
	if *namespace != "" {
		namespaces := strings.Split(strings.ReplaceAll(*namespace, " ", ""), ",")
		logger.Infof("Controller scoped to namespaces: %s", namespaces)
	}

	// High availability setting
	if *disableHA {
		ctx = sharedmain.WithHADisabled(ctx)
	}

	// Start controllers
	sharedmain.MainWithConfig(ctx, "tekton-pruner-controller", cfg,
		tektonpruner.NewController,
		pipelinerun.NewController,
		taskrun.NewController,
	)
}
