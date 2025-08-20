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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// Configuration keys
	MetricsProtocolKey = "metrics-protocol"
	MetricsEndpointKey = "metrics-endpoint"
	MetricsEnabledKey  = "metrics.enabled"
	MetricsPathKey     = "metrics.path"
)

// ParseMetricsConfigFromConfigMap creates configuration from a ConfigMap
func ParseMetricsConfigFromConfigMap(configMap *corev1.ConfigMap) *MetricsConfig {
	config := DefaultMetricsConfig()

	if configMap == nil {
		return config
	}

	// Parse protocol
	if protocol, ok := configMap.Data[MetricsProtocolKey]; ok {
		config.Protocol = strings.ToLower(strings.TrimSpace(protocol))
	}

	// Parse endpoint
	if endpoint, ok := configMap.Data[MetricsEndpointKey]; ok {
		config.Endpoint = strings.TrimSpace(endpoint)
	}

	// Parse path
	if path, ok := configMap.Data[MetricsPathKey]; ok {
		config.Path = strings.TrimSpace(path)
	}

	// Parse enabled flag
	if enabled, ok := configMap.Data[MetricsEnabledKey]; ok {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(enabled)); err == nil {
			config.Enabled = parsed
		}
	}

	// Disable if protocol is "none"
	if config.Protocol == "none" {
		config.Enabled = false
	}

	return config
}
