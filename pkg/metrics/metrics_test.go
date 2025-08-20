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
"testing"

corev1 "k8s.io/api/core/v1"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDefaultMetricsConfig(t *testing.T) {
config := DefaultMetricsConfig()

if config.Enabled != true {
t.Errorf("Expected Enabled to be true, got %v", config.Enabled)
}

if config.Protocol != "prometheus" {
t.Errorf("Expected Protocol to be prometheus, got %v", config.Protocol)
}

if config.Endpoint != ":9090" {
t.Errorf("Expected Endpoint to be :9090, got %v", config.Endpoint)
}
}

func TestParseMetricsConfigFromConfigMap(t *testing.T) {
tests := []struct {
name     string
configMap *corev1.ConfigMap
expected *MetricsConfig
}{
{
name:     "nil configmap",
configMap: nil,
expected: DefaultMetricsConfig(),
},
{
name: "disabled metrics",
configMap: &corev1.ConfigMap{
ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
Data: map[string]string{
MetricsProtocolKey: "none",
},
},
expected: &MetricsConfig{
Enabled:  false,
Protocol: "none",
Endpoint: ":9090",
Path:     "/metrics",
},
},
{
name: "custom configuration",
configMap: &corev1.ConfigMap{
ObjectMeta: metav1.ObjectMeta{Name: "test-config"},
Data: map[string]string{
MetricsProtocolKey: "prometheus",
MetricsEndpointKey: ":8080",
MetricsPathKey:     "/custom-metrics",
MetricsEnabledKey:  "true",
},
},
expected: &MetricsConfig{
Enabled:  true,
Protocol: "prometheus",
Endpoint: ":8080",
Path:     "/custom-metrics",
},
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
config := ParseMetricsConfigFromConfigMap(tt.configMap)

if config.Enabled != tt.expected.Enabled {
t.Errorf("Expected Enabled %v, got %v", tt.expected.Enabled, config.Enabled)
}
if config.Protocol != tt.expected.Protocol {
t.Errorf("Expected Protocol %v, got %v", tt.expected.Protocol, config.Protocol)
}
if config.Endpoint != tt.expected.Endpoint {
t.Errorf("Expected Endpoint %v, got %v", tt.expected.Endpoint, config.Endpoint)
}
if config.Path != tt.expected.Path {
t.Errorf("Expected Path %v, got %v", tt.expected.Path, config.Path)
}
})
}
}
