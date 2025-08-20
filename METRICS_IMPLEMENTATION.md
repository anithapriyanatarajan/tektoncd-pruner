# OpenTelemetry Metrics Implementation for Tekton Pruner

This comprehensive implementation provides a complete observability solution for the Tekton Pruner using OpenTelemetry with Prometheus backend support.

## 🎯 Implementation Overview

### What's Provided

1. **Complete Metrics Infrastructure**:
   - OpenTelemetry-based metrics collection
   - Prometheus exporter with HTTP endpoint
   - Configurable via Kubernetes ConfigMaps
   - Graceful shutdown handling

2. **Comprehensive Metrics Coverage**:
   - Resource processing counters
   - Performance timing histograms
   - Error tracking and classification
   - Resource state monitoring
   - Resource age analysis

3. **Production-Ready Features**:
   - Singleton pattern for thread safety
   - Configuration hot-reloading
   - Proper error handling
   - Performance optimized

## 📁 File Structure

```
pkg/metrics/
├── metrics.go          # Core metrics definitions and recorder
├── setup.go           # OpenTelemetry setup and lifecycle management
├── config.go          # ConfigMap-based configuration management
└── metrics_test.go    # Comprehensive test suite

cmd/controller/
├── main.go                # Enhanced with basic metrics initialization  
└── main_enhanced.go       # Advanced example with configmap watching

config/
└── config-observability.yaml  # Enhanced ConfigMap with all options

examples/
└── metrics_usage.go           # Comprehensive usage examples

docs/
├── observability-integration-guide.md  # Complete implementation guide
├── grafana-dashboard.json              # Sample Grafana dashboard
└── prometheus-alerts.yaml             # Production alerting rules
```

## 🚀 Quick Start

### 1. Initialize Metrics in Your Controller

```go
import "github.com/openshift-pipelines/tektoncd-pruner/pkg/metrics"

func main() {
    ctx := signals.NewContext()
    
    // Initialize metrics with default configuration
    exporter := metrics.GetExporter()
    config := metrics.DefaultMetricsConfig()
    if err := exporter.Initialize(ctx, config); err != nil {
        logger.Errorf("Failed to initialize metrics: %v", err)
    }
    
    // Your controller setup...
}
```

### 2. Instrument Your Reconciler

```go
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
    recorder := metrics.GetRecorder()
    
    // Start timing
    timer := recorder.NewTimer(
        metrics.ResourceAttributes(metrics.ResourceTypePipelineRun, req.Namespace)...,
    )
    defer timer.RecordReconciliationDuration(ctx)
    
    // Track processing
    recorder.RecordResourceProcessed(ctx, 
        metrics.ResourceTypePipelineRun, 
        req.Namespace, 
        metrics.StatusSuccess)
    
    // Your business logic...
    
    return reconcile.Result{}, nil
}
```

### 3. Configure via ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-observability-tekton-pruner
  namespace: tekton-pipelines
data:
  metrics-protocol: "prometheus"
  metrics-endpoint: ":9090"
  metrics.enabled: "true"
  metrics.path: "/metrics"
```

## 📊 Available Metrics

| Metric | Type | Purpose |
|--------|------|---------|
| `tektoncd_pruner_resources_processed_total` | Counter | Track resource processing |
| `tektoncd_pruner_resources_deleted_total` | Counter | Track successful deletions |
| `tektoncd_pruner_resources_errors_total` | Counter | Monitor error rates |
| `tektoncd_pruner_reconciliation_duration_seconds` | Histogram | Performance monitoring |
| `tektoncd_pruner_active_resources_count` | UpDownCounter | Current system state |
| `tektoncd_pruner_resource_age_at_deletion_seconds` | Histogram | Resource lifecycle analysis |

## 🔧 Configuration Options

### Basic Configuration
- `metrics-protocol`: `prometheus`, `otlp`, or `none`
- `metrics-endpoint`: Port and interface (e.g., `:9090`)
- `metrics.enabled`: Enable/disable metrics collection

### Advanced Configuration
- `metrics.path`: HTTP endpoint path (default: `/metrics`)
- `metrics.runtime-enabled`: Include Go runtime metrics
- `metrics.shutdown-timeout`: Graceful shutdown timeout

## 📈 Monitoring & Alerting

### Grafana Dashboard
Import the provided dashboard from `docs/grafana-dashboard.json` for:
- Resource processing rates
- Error rate monitoring
- Performance metrics
- Resource state visualization

### Prometheus Alerts
Deploy the alerting rules from `docs/prometheus-alerts.yaml` to monitor:
- High error rates
- Slow reconciliation
- System health issues
- Resource processing anomalies

## 🧪 Testing

Run the comprehensive test suite:

```bash
cd pkg/metrics
go test -v ./...
```

Run performance benchmarks:

```bash
go test -bench=. -benchmem ./...
```

## 🔍 Verification

### Check Metrics Endpoint
```bash
curl http://localhost:9090/metrics | grep tektoncd_pruner
```

### Validate Configuration
```bash
kubectl get configmap config-observability-tekton-pruner -n tekton-pipelines -o yaml
```

## 🚨 Troubleshooting

### Common Issues

1. **Metrics not appearing**:
   - Verify exporter initialization in logs
   - Check ConfigMap configuration
   - Ensure port is not blocked

2. **Configuration not loading**:
   - Verify ConfigMap name and namespace
   - Check controller has proper RBAC permissions
   - Review controller logs for errors

3. **Performance issues**:
   - Monitor metrics recording overhead in benchmarks
   - Adjust histogram bucket boundaries if needed
   - Consider sampling for high-volume metrics

### Debug Mode

Enable debug logging:
```yaml
# In config-logging.yaml
data:
  loglevel.controller: "debug"
```

## 🔮 Future Enhancements

- **OTLP Support**: Add OTLP exporter for modern observability platforms
- **Custom Metrics**: Allow user-defined metrics via configuration
- **Tracing Integration**: Add distributed tracing support
- **Advanced Dashboards**: Create role-specific dashboards

## 📚 Documentation

- [Complete Integration Guide](docs/observability-integration-guide.md)
- [Usage Examples](examples/metrics_usage.go)
- [Current Metrics Documentation](docs/metrics.md)

## 🤝 Contributing

When adding new metrics:
1. Define constants in `pkg/metrics/metrics.go`
2. Add recording methods to the Recorder
3. Update documentation and examples
4. Add appropriate tests
5. Update Grafana dashboards and alerts

## ✅ Migration Checklist

If migrating from existing metrics:

- [ ] Update imports to use OpenTelemetry packages
- [ ] Replace direct instrumentation with Recorder pattern
- [ ] Update main.go to initialize metrics exporter
- [ ] Configure observability ConfigMap
- [ ] Update monitoring dashboards
- [ ] Test metrics endpoint
- [ ] Verify graceful shutdown
- [ ] Update documentation

This implementation provides a solid foundation for production-grade observability in your Tekton Pruner project while following Kubernetes and OpenTelemetry best practices.
