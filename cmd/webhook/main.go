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

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	"github.com/openshift-pipelines/tektoncd-pruner/pkg/config"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

type WebhookServer struct {
	server *http.Server
}

func main() {
	certPath := os.Getenv("TLS_CERT_FILE")
	keyPath := os.Getenv("TLS_PRIVATE_KEY_FILE")

	if certPath == "" || keyPath == "" {
		klog.Fatal("TLS_CERT_FILE and TLS_PRIVATE_KEY_FILE must be set")
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		klog.Fatalf("Failed to load key pair: %v", err)
	}

	server := &WebhookServer{
		server: &http.Server{
			Addr: ":8443",
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS13,
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/validate-configmap", server.validateConfigMap)
	server.server.Handler = mux

	klog.Info("Starting webhook server...")
	if err := server.server.ListenAndServeTLS("", ""); err != nil {
		klog.Fatalf("Failed to start webhook server: %v", err)
	}
}

func (ws *WebhookServer) validateConfigMap(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	var admissionResponse *admissionv1.AdmissionResponse
	ar := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		klog.Errorf("Could not unmarshal admission review: %v", err)
		admissionResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = ws.validateConfigMapAdmission(ar.Request)
	}

	admissionReview := admissionv1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	respBytes, _ := json.Marshal(admissionReview)
	w.Header().Set("Content-Type", "application/json")
	// #nosec G104 - Writing JSON response for admission webhook, not HTML content
	if _, err := w.Write(respBytes); err != nil {
		klog.Errorf("Could not write response: %v", err)
	}
}

func (ws *WebhookServer) validateConfigMapAdmission(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	var configMap corev1.ConfigMap
	if err := json.Unmarshal(req.Object.Raw, &configMap); err != nil {
		klog.Errorf("Could not unmarshal configmap: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	// Only validate tekton-pruner-default-spec ConfigMap
	if configMap.Name != "tekton-pruner-default-spec" || configMap.Namespace != "tekton-pipelines" {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	// Validate the ConfigMap
	if err := validatePrunerConfigMap(&configMap); err != nil {
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}

func validatePrunerConfigMap(cm *corev1.ConfigMap) error {
	if cm.Data == nil {
		return fmt.Errorf("configmap data is required")
	}

	globalConfigData, exists := cm.Data["global-config"]
	if !exists || globalConfigData == "" {
		return fmt.Errorf("global-config data field is required")
	}

	// Parse the YAML configuration
	var prunerConfig config.PrunerConfig
	if err := yaml.Unmarshal([]byte(globalConfigData), &prunerConfig); err != nil {
		return fmt.Errorf("failed to parse global-config YAML: %v", err)
	}

	// Validate field types and values
	if err := validatePrunerConfigFields(&prunerConfig); err != nil {
		return fmt.Errorf("configuration validation failed: %v", err)
	}

	return nil
}

func validatePrunerConfigFields(cfg *config.PrunerConfig) error {
	// Validate enforcedConfigLevel
	if cfg.EnforcedConfigLevel != nil {
		switch *cfg.EnforcedConfigLevel {
		case config.EnforcedConfigLevelGlobal, config.EnforcedConfigLevelNamespace, config.EnforcedConfigLevelResource:
			// Valid values
		default:
			return fmt.Errorf("enforcedConfigLevel must be one of: global, namespace, resource")
		}
	}

	// Validate numeric fields are non-negative
	if cfg.TTLSecondsAfterFinished != nil && *cfg.TTLSecondsAfterFinished < 0 {
		return fmt.Errorf("ttlSecondsAfterFinished must be non-negative")
	}

	if cfg.SuccessfulHistoryLimit != nil && *cfg.SuccessfulHistoryLimit < 0 {
		return fmt.Errorf("successfulHistoryLimit must be non-negative")
	}

	if cfg.FailedHistoryLimit != nil && *cfg.FailedHistoryLimit < 0 {
		return fmt.Errorf("failedHistoryLimit must be non-negative")
	}

	if cfg.HistoryLimit != nil && *cfg.HistoryLimit < 0 {
		return fmt.Errorf("historyLimit must be non-negative")
	}

	// Validate namespace configurations
	for nsName, nsSpec := range cfg.Namespaces {
		if err := validateNamespaceSpec(nsName, &nsSpec); err != nil {
			return fmt.Errorf("namespace %s validation failed: %v", nsName, err)
		}
	}

	return nil
}

func validateNamespaceSpec(nsName string, nsSpec *config.NamespaceSpec) error {
	// Validate numeric fields
	if nsSpec.TTLSecondsAfterFinished != nil && *nsSpec.TTLSecondsAfterFinished < 0 {
		return fmt.Errorf("ttlSecondsAfterFinished must be non-negative")
	}

	if nsSpec.SuccessfulHistoryLimit != nil && *nsSpec.SuccessfulHistoryLimit < 0 {
		return fmt.Errorf("successfulHistoryLimit must be non-negative")
	}

	if nsSpec.FailedHistoryLimit != nil && *nsSpec.FailedHistoryLimit < 0 {
		return fmt.Errorf("failedHistoryLimit must be non-negative")
	}

	if nsSpec.HistoryLimit != nil && *nsSpec.HistoryLimit < 0 {
		return fmt.Errorf("historyLimit must be non-negative")
	}

	// Validate resource specs
	for i, resourceSpec := range nsSpec.PipelineRuns {
		if err := validateResourceSpec(fmt.Sprintf("pipelineRuns[%d]", i), &resourceSpec); err != nil {
			return err
		}
	}

	for i, resourceSpec := range nsSpec.TaskRuns {
		if err := validateResourceSpec(fmt.Sprintf("taskRuns[%d]", i), &resourceSpec); err != nil {
			return err
		}
	}

	return nil
}

func validateResourceSpec(fieldPath string, resourceSpec *config.ResourceSpec) error {
	// Validate numeric fields
	if resourceSpec.TTLSecondsAfterFinished != nil && *resourceSpec.TTLSecondsAfterFinished < 0 {
		return fmt.Errorf("%s.ttlSecondsAfterFinished must be non-negative", fieldPath)
	}

	if resourceSpec.SuccessfulHistoryLimit != nil && *resourceSpec.SuccessfulHistoryLimit < 0 {
		return fmt.Errorf("%s.successfulHistoryLimit must be non-negative", fieldPath)
	}

	if resourceSpec.FailedHistoryLimit != nil && *resourceSpec.FailedHistoryLimit < 0 {
		return fmt.Errorf("%s.failedHistoryLimit must be non-negative", fieldPath)
	}

	if resourceSpec.HistoryLimit != nil && *resourceSpec.HistoryLimit < 0 {
		return fmt.Errorf("%s.historyLimit must be non-negative", fieldPath)
	}

	// Validate enforcedConfigLevel
	if resourceSpec.EnforcedConfigLevel != nil {
		switch *resourceSpec.EnforcedConfigLevel {
		case config.EnforcedConfigLevelGlobal, config.EnforcedConfigLevelNamespace, config.EnforcedConfigLevelResource:
			// Valid values
		default:
			return fmt.Errorf("%s.enforcedConfigLevel must be one of: global, namespace, resource", fieldPath)
		}
	}

	return nil
}
