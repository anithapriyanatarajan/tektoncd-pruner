package helper

import (
	"sync"

	//tektonprunerv1alpha1 "github.com/openshift-pipelines/tektoncd-pruner/pkg/apis/tektonpruner/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// for internal use
// to manage different resources and different fields
type PrunerResourceType string
type PrunerFieldType string
type EnforcedConfigLevel string

const (
	PrunerResourceTypePipelineRun PrunerResourceType = "pipelinerun"
	PrunerResourceTypeTaskRun     PrunerResourceType = "taskrun"

	PrunerFieldTypeTTLSecondsAfterFinished PrunerFieldType = "ttlSecondsAfterFinished"
	PrunerFieldTypeSuccessfulHistoryLimit  PrunerFieldType = "successfulHistoryLimit"
	PrunerFieldTypeFailedHistoryLimit      PrunerFieldType = "failedHistoryLimit"

	EnforcedConfigLevelGlobal    EnforcedConfigLevel = "global"
	EnforcedConfigLevelNamespace EnforcedConfigLevel = "namespace"
	EnforcedConfigLevelResource  EnforcedConfigLevel = "resource"
)

// Run represents a configuration for selecting a PipelineRun or TaskRun.
type ResourceSpec struct {
	Selector Selector `yaml:"selector"`
}

// used to hold the config of a specific pr/tr
type Selector struct {
    Name      string                 `yaml:"name,omitempty"` //indicates the pipelinename for pipelines and taskname for taskruns
	//PipelineName      string                 `yaml:"pipelineName,omitempty"`
	//TaskName          string                 `yaml:"taskName,omitempty"`
	MatchLabels       map[string]string      `yaml:"matchLabels,omitempty"`
	MatchAnnotations  map[string]string      `yaml:"matchAnnotations,omitempty"`
	TTLSecondsAfterFinished *int32              `yaml:"ttlSecondsAfterFinished"`
	SuccessfulHistoryLimit  *int32              `yaml:"successfulHistoryLimit"`
	FailedHistoryLimit      *int32              `yaml:"failedHistoryLimit"`
	EnforcedConfigLevel     EnforcedConfigLevel `yaml:"enforcedConfigLevel"`
}

// used to hold the config of a specific namespace
type NamespaceSpec struct {
	// EnforcedConfigLevel allowed values: global, namespace, resource (default: resource)
	EnforcedConfigLevel     EnforcedConfigLevel `yaml:"enforcedConfigLevel"`
	TTLSecondsAfterFinished *int32                                    `yaml:"ttlSecondsAfterFinished"`
	SuccessfulHistoryLimit  *int32                                    `yaml:"successfulHistoryLimit"`
	FailedHistoryLimit      *int32                                    `yaml:"failedHistoryLimit"`
	HistoryLimit            *int32                                    `yaml:"historyLimit"`
	PipelineRuns               []ResourceSpec      `yaml:"pipelineruns"`
	TaskRuns                   []ResourceSpec       `yaml:"taskruns"`
}



// used to hold the config of namespaces
// and global config
type PrunerConfig struct {
	// EnforcedConfigLevel allowed values: global, namespace, resource (default: resource)
	EnforcedConfigLevel     EnforcedConfigLevel `yaml:"enforcedConfigLevel"`
	TTLSecondsAfterFinished *int32                                    `yaml:"ttlSecondsAfterFinished"`
	SuccessfulHistoryLimit  *int32                                    `yaml:"successfulHistoryLimit"`
	FailedHistoryLimit      *int32                                    `yaml:"failedHistoryLimit"`
	HistoryLimit            *int32                                    `yaml:"historyLimit"`
	Namespaces              map[string]NamespaceSpec             `yaml:"namespaces"`
}

// defines the store structure
// holds config from ConfigMap (global config) and config from namespaces (namespaced config)
type prunerConfigStore struct {
	mutex            sync.RWMutex
	globalConfig     PrunerConfig
	namespacedConfig map[string]NamespaceSpec
}

var (
	// store to manage pruner config
	// singleton instance
	PrunerConfigStore = prunerConfigStore{mutex: sync.RWMutex{}}
)

// loads config from configMap (global-config)
// should be called on startup and if there is a change detected on the ConfigMap
func (ps *prunerConfigStore) LoadGlobalConfig(configMap *corev1.ConfigMap) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	globalConfig := &PrunerConfig{}
	if configMap.Data != nil && configMap.Data[PrunerGlobalConfigKey] != "" {
		err := yaml.Unmarshal([]byte(configMap.Data[PrunerGlobalConfigKey]), globalConfig)
		if err != nil {
			return err
		}
	}

	ps.globalConfig = *globalConfig

	if ps.globalConfig.Namespaces == nil {
		ps.globalConfig.Namespaces = map[string]NamespaceSpec{}
	}

	if ps.namespacedConfig == nil {
		ps.namespacedConfig = map[string]NamespaceSpec{}
	}

	return nil
}

/*
func (ps *prunerConfigStore) UpdateNamespacedSpec(prunerCR *tektonprunerv1alpha1.TektonPruner) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	namespace := prunerCR.Namespace

	// update in the local store
	namespacedSpec := NamespaceSpec{
		TTLSecondsAfterFinished: prunerCR.Spec.TTLSecondsAfterFinished,
		Pipelines:               prunerCR.Spec.Pipelines,
		Tasks:                   prunerCR.Spec.Tasks,
	}
	ps.namespacedConfig[namespace] = namespacedSpec
}
*/

func (ps *prunerConfigStore) DeleteNamespacedSpec(namespace string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	delete(ps.namespacedConfig, namespace)
}

func getFromPrunerConfigResourceLevel(namespacesSpec map[string]NamespaceSpec, namespace, name string, resourceType PrunerResourceType, fieldType PrunerFieldType) *int32 {
	NamespaceSpec, found := namespacesSpec[namespace]
	if !found {
		return nil
	}

	var resourceSpecs []ResourceSpec

	switch resourceType {
	case PrunerResourceTypePipelineRun:
		resourceSpecs = NamespaceSpec.PipelineRuns

	case PrunerResourceTypeTaskRun:
		resourceSpecs = NamespaceSpec.TaskRuns
	}

	for _, resourceSpec := range resourceSpecs {
		if resourceSpec.Selector.Name == name {
			switch fieldType {
			case PrunerFieldTypeTTLSecondsAfterFinished:
				return resourceSpec.Selector.TTLSecondsAfterFinished

			case PrunerFieldTypeSuccessfulHistoryLimit:
				return resourceSpec.Selector.SuccessfulHistoryLimit

			case PrunerFieldTypeFailedHistoryLimit:
				return resourceSpec.Selector.FailedHistoryLimit
			}
		}
	}
	return nil
}

func getResourceFieldData(namespacedSpec map[string]NamespaceSpec, globalSpec PrunerConfig, namespace, name string, resourceType PrunerResourceType, fieldType PrunerFieldType, enforcedConfigLevel EnforcedConfigLevel) *int32 {
	var ttl *int32

	switch enforcedConfigLevel {
	case EnforcedConfigLevelResource:
		// get from namespaced spec, resource level
		ttl = getFromPrunerConfigResourceLevel(namespacedSpec, namespace, name, resourceType, fieldType)

		fallthrough

	case EnforcedConfigLevelNamespace:
		if ttl == nil {
			// get it from namespace spec, root level
			spec, found := namespacedSpec[namespace]
			if found {
				switch fieldType {
				case PrunerFieldTypeTTLSecondsAfterFinished:
					ttl = spec.TTLSecondsAfterFinished

				case PrunerFieldTypeSuccessfulHistoryLimit:
					ttl = spec.SuccessfulHistoryLimit

				case PrunerFieldTypeFailedHistoryLimit:
					ttl = spec.FailedHistoryLimit
				}
			}
		}
		fallthrough

	case EnforcedConfigLevelGlobal:
		if ttl == nil {
			// get from global spec, resource level
			ttl = getFromPrunerConfigResourceLevel(globalSpec.Namespaces, namespace, name, resourceType, fieldType)
		}

		if ttl == nil {
			// get it from global spec, namespace root level
			spec, found := globalSpec.Namespaces[namespace]
			if found {
				switch fieldType {
				case PrunerFieldTypeTTLSecondsAfterFinished:
					ttl = spec.TTLSecondsAfterFinished

				case PrunerFieldTypeSuccessfulHistoryLimit:
					ttl = spec.SuccessfulHistoryLimit

				case PrunerFieldTypeFailedHistoryLimit:
					ttl = spec.FailedHistoryLimit
				}
			}
		}

		if ttl == nil {
			// get it from global spec, root level
			switch fieldType {
			case PrunerFieldTypeTTLSecondsAfterFinished:
				ttl = globalSpec.TTLSecondsAfterFinished

			case PrunerFieldTypeSuccessfulHistoryLimit:
				ttl = globalSpec.SuccessfulHistoryLimit

			case PrunerFieldTypeFailedHistoryLimit:
				ttl = globalSpec.FailedHistoryLimit
			}
		}

	}

	return ttl
}

func (ps *prunerConfigStore) GetEnforcedConfigLevelFromNamespaceSpec(namespacesSpec map[string]NamespaceSpec, namespace, name string, resourceType PrunerResourceType) EnforcedConfigLevel {
	var enforcedConfigLevel EnforcedConfigLevel
	var resourceSpecs []ResourceSpec
	var namespaceSpec NamespaceSpec
	var found bool

	namespaceSpec, found = ps.globalConfig.Namespaces[namespace]
	if found {
		switch resourceType {
		case PrunerResourceTypePipelineRun:
			resourceSpecs = namespaceSpec.PipelineRuns

		case PrunerResourceTypeTaskRun:
			resourceSpecs = namespaceSpec.TaskRuns
		}
		for _, resourceSpec := range resourceSpecs {
			if resourceSpec.Selector.Name == name {
				// if found on resource level
				enforcedConfigLevel = resourceSpec.Selector.EnforcedConfigLevel
				if enforcedConfigLevel != "" {
					return enforcedConfigLevel
				}
				break
			}
		}

		// get it from namespace root level
		enforcedConfigLevel = namespaceSpec.EnforcedConfigLevel
		if enforcedConfigLevel != "" {
			return enforcedConfigLevel
		}
	}
	return ""
}

func (ps *prunerConfigStore) getEnforcedConfigLevel(namespace, name string, resourceType PrunerResourceType) EnforcedConfigLevel {
	var enforcedConfigLevel EnforcedConfigLevel

	// get it from global spec (order: resource level, namespace root level)
	enforcedConfigLevel = ps.GetEnforcedConfigLevelFromNamespaceSpec(ps.globalConfig.Namespaces, namespace, name, resourceType)
	if enforcedConfigLevel != "" {
		return enforcedConfigLevel
	}

	// get it from global spec, root level
	enforcedConfigLevel = ps.globalConfig.EnforcedConfigLevel
	if enforcedConfigLevel != "" {
		return enforcedConfigLevel
	}

	// get it from namespace spec (order: resource level, root level)
	enforcedConfigLevel = ps.GetEnforcedConfigLevelFromNamespaceSpec(ps.namespacedConfig, namespace, name, resourceType)
	if enforcedConfigLevel != "" {
		return enforcedConfigLevel
	}

	// default level, if no where specified
	return EnforcedConfigLevelResource
}

func (ps *prunerConfigStore) GetPipelineEnforcedConfigLevel(namespace, name string) EnforcedConfigLevel {
	return ps.getEnforcedConfigLevel(namespace, name, PrunerResourceTypePipelineRun)
}

func (ps *prunerConfigStore) GetTaskEnforcedConfigLevel(namespace, name string) EnforcedConfigLevel {
	return ps.getEnforcedConfigLevel(namespace, name, PrunerResourceTypeTaskRun)
}

func (ps *prunerConfigStore) GetPipelineTTLSecondsAfterFinished(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetPipelineEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypePipelineRun, PrunerFieldTypeTTLSecondsAfterFinished, enforcedConfigLevel)
}

func (ps *prunerConfigStore) GetPipelineSuccessHistoryLimitCount(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetPipelineEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypePipelineRun, PrunerFieldTypeSuccessfulHistoryLimit, enforcedConfigLevel)
}

func (ps *prunerConfigStore) GetPipelineFailedHistoryLimitCount(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetPipelineEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypePipelineRun, PrunerFieldTypeFailedHistoryLimit, enforcedConfigLevel)
}

func (ps *prunerConfigStore) GetTaskTTLSecondsAfterFinished(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetTaskEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypeTaskRun, PrunerFieldTypeTTLSecondsAfterFinished, enforcedConfigLevel)
}

func (ps *prunerConfigStore) GetTaskSuccessHistoryLimitCount(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetTaskEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypeTaskRun, PrunerFieldTypeSuccessfulHistoryLimit, enforcedConfigLevel)
}

func (ps *prunerConfigStore) GetTaskFailedHistoryLimitCount(namespace, name string) *int32 {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	enforcedConfigLevel := ps.GetTaskEnforcedConfigLevel(namespace, name)
	return getResourceFieldData(ps.namespacedConfig, ps.globalConfig, namespace, name, PrunerResourceTypeTaskRun, PrunerFieldTypeFailedHistoryLimit, enforcedConfigLevel)
}
