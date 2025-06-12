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

package config

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestPrunerConfigUnmarshalYAML_StringToInt(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		expectError bool
		validate    func(*PrunerConfig) error
	}{
		{
			name: "historyLimit as string",
			yamlContent: `
enforcedConfigLevel: global
historyLimit: "100"
`,
			expectError: false,
			validate: func(pc *PrunerConfig) error {
				if pc.HistoryLimit == nil || *pc.HistoryLimit != 100 {
					t.Errorf("Expected historyLimit to be 100, got %v", pc.HistoryLimit)
				}
				return nil
			},
		},
		{
			name: "historyLimit as int",
			yamlContent: `
enforcedConfigLevel: global
historyLimit: 100
`,
			expectError: false,
			validate: func(pc *PrunerConfig) error {
				if pc.HistoryLimit == nil || *pc.HistoryLimit != 100 {
					t.Errorf("Expected historyLimit to be 100, got %v", pc.HistoryLimit)
				}
				return nil
			},
		},
		{
			name: "multiple fields with string values",
			yamlContent: `
enforcedConfigLevel: global
historyLimit: "100"
ttlSecondsAfterFinished: "300"
successfulHistoryLimit: "5"
failedHistoryLimit: "3"
`,
			expectError: false,
			validate: func(pc *PrunerConfig) error {
				if pc.HistoryLimit == nil || *pc.HistoryLimit != 100 {
					t.Errorf("Expected historyLimit to be 100, got %v", pc.HistoryLimit)
				}
				if pc.TTLSecondsAfterFinished == nil || *pc.TTLSecondsAfterFinished != 300 {
					t.Errorf("Expected ttlSecondsAfterFinished to be 300, got %v", pc.TTLSecondsAfterFinished)
				}
				if pc.SuccessfulHistoryLimit == nil || *pc.SuccessfulHistoryLimit != 5 {
					t.Errorf("Expected successfulHistoryLimit to be 5, got %v", pc.SuccessfulHistoryLimit)
				}
				if pc.FailedHistoryLimit == nil || *pc.FailedHistoryLimit != 3 {
					t.Errorf("Expected failedHistoryLimit to be 3, got %v", pc.FailedHistoryLimit)
				}
				return nil
			},
		},
		{
			name: "invalid string value",
			yamlContent: `
enforcedConfigLevel: global
historyLimit: "invalid"
`,
			expectError: true,
			validate:    func(pc *PrunerConfig) error { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config PrunerConfig
			err := yaml.Unmarshal([]byte(tt.yamlContent), &config)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && err == nil {
				if validationErr := tt.validate(&config); validationErr != nil {
					t.Errorf("Validation failed: %v", validationErr)
				}
			}
		})
	}
}

func TestParseIntField(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		fieldName   string
		expected    *int32
		expectError bool
	}{
		{
			name:      "string number",
			value:     "123",
			fieldName: "test",
			expected:  func() *int32 { v := int32(123); return &v }(),
		},
		{
			name:      "int",
			value:     123,
			fieldName: "test",
			expected:  func() *int32 { v := int32(123); return &v }(),
		},
		{
			name:      "int32",
			value:     int32(123),
			fieldName: "test",
			expected:  func() *int32 { v := int32(123); return &v }(),
		},
		{
			name:      "nil",
			value:     nil,
			fieldName: "test",
			expected:  nil,
		},
		{
			name:        "invalid string",
			value:       "invalid",
			fieldName:   "test",
			expectError: true,
		},
		{
			name:        "invalid type",
			value:       struct{}{},
			fieldName:   "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntField(tt.value, tt.fieldName)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if tt.expected == nil && result != nil {
					t.Errorf("Expected nil but got %v", result)
				}
				if tt.expected != nil && result == nil {
					t.Errorf("Expected %v but got nil", *tt.expected)
				}
				if tt.expected != nil && result != nil && *tt.expected != *result {
					t.Errorf("Expected %v but got %v", *tt.expected, *result)
				}
			}
		})
	}
}
