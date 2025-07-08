package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/openshift-pipelines/tektoncd-pruner/pkg/observability"
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

// main function of the program
func main() {
	// Define command-line flags
	flag.IntVar(&controller.DefaultThreadsPerController, "threads-per-controller", controller.DefaultThreadsPerController, "Threads (goroutines) to create per controller")
	namespace := flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")
	disableHighAvailability := flag.Bool("disable-ha", true, "Whether to disable high-availability functionality for this component.")
	flag.Parse()

	// Parse and get REST config
	cfg := injection.ParseAndGetRESTConfigOrDie()

	// Set QPS and Burst settings
	if cfg.QPS == 0 {
		cfg.QPS = 2 * rest.DefaultQPS
	}
	if cfg.Burst == 0 {
		cfg.Burst = rest.DefaultBurst
	}

	// Multiply by 2 for number of controllers
	cfg.QPS = 2 * cfg.QPS
	cfg.Burst = 2 * cfg.Burst

	// Set up logging
	ctx := signals.NewContext()
	logger := logging.FromContext(ctx)

	// Initialize observability
	observabilityConfig := observability.LoadConfigFromEnv()
	observabilitySetup, err := observability.SetupObservability(ctx, observabilityConfig)
	if err != nil {
		logger.Fatalf("Failed to setup observability: %v", err)
	}

	// Setup graceful shutdown for observability
	defer func() {
		// Give observability 10 seconds to shutdown gracefully
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := observabilitySetup.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Error during observability shutdown: %v", err)
		}
	}()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Received shutdown signal, starting graceful shutdown...")
		os.Exit(0)
	}()

	// Initialize global metrics and tracing helpers
	if observabilityConfig.MetricsEnabled {
		if err := observability.InitializeGlobalMetrics(ctx, observabilitySetup.GetMeterProvider()); err != nil {
			logger.Fatalf("Failed to initialize global metrics: %v", err)
		}
		logger.Info("Global metrics initialized")
	}

	if observabilityConfig.TracingEnabled {
		observability.InitializeGlobalTracingHelper(ctx, observabilitySetup.GetTracerProvider())
		logger.Info("Global tracing helper initialized")
	}

	// Start metrics server if enabled
	if observabilityConfig.MetricsEnabled && observabilityConfig.PrometheusEnabled {
		go func() {
			mux := http.NewServeMux()

			// Get the metrics handler from observability setup
			metricsHandler := observabilitySetup.GetMetricsHandler()
			if metricsHandler != nil {
				mux.Handle("/metrics", metricsHandler)
			} else {
				// Use default Prometheus handler for newer OpenTelemetry versions
				mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Use the default Prometheus registry
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("# Metrics endpoint - OpenTelemetry metrics available\n"))
				}))
			}

			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
			mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Ready"))
			})

			server := &http.Server{
				Addr:    fmt.Sprintf(":%d", observabilityConfig.MetricsPort),
				Handler: mux,
			}

			logger.Infof("Starting metrics server on port %d", observabilityConfig.MetricsPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Errorf("Metrics server error: %v", err)
			}
		}()
	}

	// Initialize Knative metrics for backward compatibility
	if err := observability.InitializeKnativeMetrics(ctx); err != nil {
		logger.Errorf("Failed to initialize Knative metrics: %v", err)
		// Don't fail here, as this is for backward compatibility
	}

	// Add namespaces
	var namespaces []string
	if *namespace != "" {
		namespaces = strings.Split(strings.ReplaceAll(*namespace, " ", ""), ",")
		logger.Infof("controller is scoped to the following namespaces: %s\n", namespaces)
	}

	// Add High Availability flag
	if *disableHighAvailability {
		ctx = sharedmain.WithHADisabled(ctx)
	}

	logger.Info("Starting tekton-pruner-controller with observability enabled")

	// Use sharedmain to handle controller lifecycle
	sharedmain.MainWithConfig(ctx, "tekton-pruner-controller", cfg,
		tektonpruner.NewController,
		pipelinerun.NewController,
		taskrun.NewController,
	)
}
