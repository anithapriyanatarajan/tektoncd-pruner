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
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"knative.dev/pkg/logging"
)

// MetricsConfig holds basic metrics configuration
type MetricsConfig struct {
	Enabled  bool
	Protocol string
	Endpoint string
	Path     string
}

// DefaultMetricsConfig returns default configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:  true,
		Protocol: "prometheus",
		Endpoint: ":9090",
		Path:     "/metrics",
	}
}

// MetricsExporter handles metrics setup
type MetricsExporter struct {
	config        *MetricsConfig
	meterProvider *sdkmetric.MeterProvider
	server        *http.Server
	mu            sync.RWMutex
	isInitialized bool
}

var (
	globalExporter *MetricsExporter
	exporterOnce   sync.Once
)

// GetExporter returns the global metrics exporter
func GetExporter() *MetricsExporter {
	exporterOnce.Do(func() {
		globalExporter = &MetricsExporter{}
	})
	return globalExporter
}

// Initialize sets up metrics with the given configuration
func (e *MetricsExporter) Initialize(ctx context.Context, config *MetricsConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isInitialized {
		return nil
	}

	logger := logging.FromContext(ctx)
	e.config = config

	if !config.Enabled || config.Protocol != "prometheus" {
		logger.Info("Metrics disabled or unsupported protocol")
		return nil
	}

	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	e.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(e.meterProvider)

	// Initialize recorder
	initializeRecorder()

	// Start HTTP server
	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.Handler())

	e.server = &http.Server{
		Addr:    config.Endpoint,
		Handler: mux,
	}

	go func() {
		logger.Infof("Starting metrics server on %s%s", config.Endpoint, config.Path)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Metrics server error: %v", err)
		}
	}()

	e.isInitialized = true
	logger.Info("Metrics initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the metrics exporter
func (e *MetricsExporter) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isInitialized {
		return nil
	}

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if e.server != nil {
		if err := e.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	if e.meterProvider != nil {
		if err := e.meterProvider.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	e.isInitialized = false
	return nil
}

// IsInitialized returns whether the exporter is initialized
func (e *MetricsExporter) IsInitialized() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isInitialized
}

// initializeRecorder resets the recorder to use the new meter provider
func initializeRecorder() {
	once = sync.Once{}
	recorder = nil
}
