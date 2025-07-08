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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
)

const (
	// Service name for OpenTelemetry
	ServiceName = "tektoncd-pruner"
	// Service version - will be populated at build time
	ServiceVersion = "dev"
	// Namespace for all metrics
	MetricsNamespace = "tektoncd_pruner"
	// Default metrics export interval
	DefaultMetricsInterval = 30 * time.Second
	// Default OTLP endpoint
	DefaultOTLPEndpoint = "http://localhost:4317"
)

// Config holds observability configuration
type Config struct {
	// Service information
	ServiceName    string
	ServiceVersion string

	// Metrics configuration
	MetricsEnabled     bool
	MetricsPort        int
	MetricsInterval    time.Duration
	PrometheusEnabled  bool
	OTLPMetricsEnabled bool

	// Tracing configuration
	TracingEnabled    bool
	TracingSampleRate float64
	OTLPTraceEnabled  bool

	// OTLP configuration
	OTLPEndpoint string
	OTLPHeaders  map[string]string
	OTLPInsecure bool

	// Resource attributes
	ResourceAttributes map[string]string
}

// DefaultConfig returns the default observability configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:        ServiceName,
		ServiceVersion:     ServiceVersion,
		MetricsEnabled:     true,
		MetricsPort:        9090,
		MetricsInterval:    DefaultMetricsInterval,
		PrometheusEnabled:  true,
		OTLPMetricsEnabled: false,
		TracingEnabled:     false,
		TracingSampleRate:  0.1,
		OTLPTraceEnabled:   false,
		OTLPEndpoint:       DefaultOTLPEndpoint,
		OTLPInsecure:       true,
		ResourceAttributes: map[string]string{
			"service.name":    ServiceName,
			"service.version": ServiceVersion,
		},
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()

	// Service configuration
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		config.ServiceName = name
		config.ResourceAttributes["service.name"] = name
	}

	if version := os.Getenv("OTEL_SERVICE_VERSION"); version != "" {
		config.ServiceVersion = version
		config.ResourceAttributes["service.version"] = version
	}

	// Metrics configuration
	if enabled := os.Getenv("METRICS_ENABLED"); enabled != "" {
		config.MetricsEnabled = parseBool(enabled, true)
	}

	if port := os.Getenv("METRICS_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.MetricsPort = p
		}
	}

	if interval := os.Getenv("METRICS_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.MetricsInterval = d
		}
	}

	if enabled := os.Getenv("PROMETHEUS_ENABLED"); enabled != "" {
		config.PrometheusEnabled = parseBool(enabled, true)
	}

	if enabled := os.Getenv("OTLP_METRICS_ENABLED"); enabled != "" {
		config.OTLPMetricsEnabled = parseBool(enabled, false)
	}

	// Tracing configuration
	if enabled := os.Getenv("TRACING_ENABLED"); enabled != "" {
		config.TracingEnabled = parseBool(enabled, false)
	}

	if rate := os.Getenv("TRACING_SAMPLE_RATE"); rate != "" {
		if r, err := strconv.ParseFloat(rate, 64); err == nil {
			config.TracingSampleRate = r
		}
	}

	if enabled := os.Getenv("OTLP_TRACE_ENABLED"); enabled != "" {
		config.OTLPTraceEnabled = parseBool(enabled, false)
	}

	// OTLP configuration
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		config.OTLPEndpoint = endpoint
	}

	if insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE"); insecure != "" {
		config.OTLPInsecure = parseBool(insecure, true)
	}

	// Parse OTLP headers
	if headers := os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"); headers != "" {
		config.OTLPHeaders = parseHeaders(headers)
	}

	// Parse resource attributes
	if attrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); attrs != "" {
		for k, v := range parseKeyValuePairs(attrs) {
			config.ResourceAttributes[k] = v
		}
	}

	return config
}

// ObservabilitySetup holds the observability setup state
type ObservabilitySetup struct {
	config          *Config
	tracerProvider  trace.TracerProvider
	meterProvider   metric.MeterProvider
	metricsShutdown func(context.Context) error
	tracingShutdown func(context.Context) error
	metricsHandler  http.Handler
	logger          *zap.SugaredLogger
}

// SetupObservability initializes OpenTelemetry with the provided configuration
func SetupObservability(ctx context.Context, config *Config) (*ObservabilitySetup, error) {
	logger := logging.FromContext(ctx)

	setup := &ObservabilitySetup{
		config: config,
		logger: logger,
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(stringMapToAttributes(config.ResourceAttributes)...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing if enabled
	if config.TracingEnabled {
		tracingShutdown, err := setup.setupTracing(ctx, res)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		setup.tracingShutdown = tracingShutdown
	}

	// Setup metrics if enabled
	if config.MetricsEnabled {
		metricsShutdown, metricsHandler, err := setup.setupMetrics(ctx, res)
		if err != nil {
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		setup.metricsShutdown = metricsShutdown
		setup.metricsHandler = metricsHandler
	}

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry observability setup completed successfully")
	return setup, nil
}

// setupTracing initializes the tracing pipeline
func (s *ObservabilitySetup) setupTracing(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if s.config.OTLPTraceEnabled {
		// Setup OTLP trace exporter
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(s.config.OTLPEndpoint),
		}

		if s.config.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		if len(s.config.OTLPHeaders) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(s.config.OTLPHeaders))
		}

		exporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
		}
	}

	// Create trace provider
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}

	if exporter != nil {
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	// Add sampling
	if s.config.TracingSampleRate > 0 {
		opts = append(opts, sdktrace.WithSampler(sdktrace.TraceIDRatioBased(s.config.TracingSampleRate)))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	s.tracerProvider = tp

	s.logger.Infof("Tracing setup completed with sample rate: %.2f", s.config.TracingSampleRate)

	return tp.Shutdown, nil
}

// setupMetrics initializes the metrics pipeline
func (s *ObservabilitySetup) setupMetrics(ctx context.Context, res *resource.Resource) (func(context.Context) error, http.Handler, error) {
	var readers []sdkmetric.Reader
	var handler http.Handler

	// Setup Prometheus exporter if enabled
	if s.config.PrometheusEnabled {
		promExporter, err := prometheus.New()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)

		// Note: In newer versions of OpenTelemetry, the Prometheus exporter
		// doesn't implement http.Handler directly. We return nil here and
		// handle metrics serving through the default Prometheus registry.
		handler = nil
	}

	// Setup OTLP metrics exporter if enabled
	if s.config.OTLPMetricsEnabled {
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(s.config.OTLPEndpoint),
		}

		if s.config.OTLPInsecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}

		if len(s.config.OTLPHeaders) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(s.config.OTLPHeaders))
		}

		exporter, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create OTLP metrics exporter: %w", err)
		}

		reader := sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(s.config.MetricsInterval))
		readers = append(readers, reader)
	}

	// Create meter provider
	providerOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	for _, reader := range readers {
		providerOpts = append(providerOpts, sdkmetric.WithReader(reader))
	}
	mp := sdkmetric.NewMeterProvider(providerOpts...)
	otel.SetMeterProvider(mp)
	s.meterProvider = mp

	s.logger.Info("Metrics setup completed")

	return mp.Shutdown, handler, nil
}

// GetTracerProvider returns the configured tracer provider
func (s *ObservabilitySetup) GetTracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

// GetMeterProvider returns the configured meter provider
func (s *ObservabilitySetup) GetMeterProvider() metric.MeterProvider {
	return s.meterProvider
}

// GetMetricsHandler returns the metrics HTTP handler (for Prometheus)
func (s *ObservabilitySetup) GetMetricsHandler() http.Handler {
	return s.metricsHandler
}

// Shutdown gracefully shuts down the observability setup
func (s *ObservabilitySetup) Shutdown(ctx context.Context) error {
	var errors []error

	if s.metricsShutdown != nil {
		if err := s.metricsShutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("metrics shutdown error: %w", err))
		}
	}

	if s.tracingShutdown != nil {
		if err := s.tracingShutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("tracing shutdown error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("observability shutdown errors: %v", errors)
	}

	s.logger.Info("Observability shutdown completed")
	return nil
}

// Utility functions

func parseBool(value string, defaultValue bool) bool {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func parseHeaders(headers string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(headers, ",")
	for _, pair := range pairs {
		if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

func parseKeyValuePairs(input string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
			result[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return result
}

func stringMapToAttributes(m map[string]string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}

// InitializeKnativeMetrics initializes the existing Knative metrics system
// This maintains compatibility with existing Knative metrics
func InitializeKnativeMetrics(ctx context.Context) error {
	logger := logging.FromContext(ctx)

	// Initialize the existing Knative metrics system with default options
	exporterOptions := metrics.ExporterOptions{}
	if err := metrics.UpdateExporter(ctx, exporterOptions, logger); err != nil {
		return fmt.Errorf("failed to update metrics exporter: %w", err)
	}

	logger.Info("Knative metrics system initialized")
	return nil
}
