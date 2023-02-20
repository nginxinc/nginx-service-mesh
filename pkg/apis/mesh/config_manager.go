package mesh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Keeps track of sidecar agent versions.
// First key is a version (x.x.x), with a map of agents that run that version.
type agentVersions map[string]map[string]struct{}

// ConfigManager manages the in-memory mesh configuration.
type ConfigManager struct {
	AgentVersions agentVersions
	meshConfig    FullMeshConfig
	sync.Mutex
}

// NewConfigManager creates a new mesh config manager.
func NewConfigManager(config FullMeshConfig) *ConfigManager {
	m := &ConfigManager{
		AgentVersions: make(agentVersions),
	}
	m.SetConfig(config)

	return m
}

// SetConfig sets the mesh config.
func (m *ConfigManager) SetConfig(config FullMeshConfig) {
	m.Lock()
	defer m.Unlock()

	m.meshConfig = config
}

// GetConfig returns mesh config.
func (m *ConfigManager) GetConfig() FullMeshConfig {
	m.Lock()
	defer m.Unlock()

	return m.meshConfig
}

// GetNamespace returns the namespace where the service mesh is deployed.
func (m *ConfigManager) GetNamespace() string {
	return m.GetConfig().Namespace
}

// GetLoadBalancingMethod returns the load balancing method string.
func (m *ConfigManager) GetLoadBalancingMethod() string {
	return m.GetConfig().NGINXLBMethod
}

// GetMtlsMode returns the mtls mode string.
func (m *ConfigManager) GetMtlsMode() string {
	return m.GetConfig().Mtls.Mode
}

// RecordAgentVersion adds an entry for a version tied to an agent.
func (m *ConfigManager) RecordAgentVersion(agent, version string) {
	m.Lock()
	defer m.Unlock()

	agentMap, ok := m.AgentVersions[version]
	if !ok {
		agentMap = make(map[string]struct{})
		m.AgentVersions[version] = agentMap
	}
	agentMap[agent] = struct{}{}
}

// DismissAgentVersion removes an entry for a version tied to an agent.
func (m *ConfigManager) DismissAgentVersion(agent string) {
	m.Lock()
	defer m.Unlock()

	for version, agentMap := range m.AgentVersions {
		if _, ok := agentMap[agent]; ok {
			delete(agentMap, agent)
			if len(agentMap) == 0 {
				delete(m.AgentVersions, version)
			}

			break
		}
	}
}

// GetAgentVersions returns a list of agent versions currently in the system.
func (m *ConfigManager) GetAgentVersions() []string {
	m.Lock()

	versions := make([]string, 0, len(m.AgentVersions))
	for v := range m.AgentVersions {
		versions = append(versions, v)
	}
	m.Unlock()
	sort.Strings(versions)

	return versions
}

// AddIgnoredNamespace adds a namespace to IgnoredNamespaces map.
func AddIgnoredNamespace(ns string) {
	IgnoredNamespaces[ns] = true
}

// GetMeshConfig attempts to get the mesh config from Kubernetes.
func GetMeshConfig(ctx context.Context, k8sClient client.Client, namespace string) (*FullMeshConfig, error) {
	var cm v1.ConfigMap
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: MeshConfigMap}, &cm); err != nil {
		return nil, fmt.Errorf("error getting mesh ConfigMap: %w", err)
	}

	var cfg FullMeshConfig
	if err := json.Unmarshal([]byte(cm.Data[MeshConfigFileName]), &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling json: %w", err)
	}

	return &cfg, nil
}
