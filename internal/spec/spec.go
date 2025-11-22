package spec

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ServiceSpec struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   MetadataSpec    `yaml:"metadata"`
	Spec       ServiceSpecBody `yaml:"spec"`
}

type MetadataSpec struct {
	Name string `yaml:"name"`
}

type ServiceSpecBody struct {
	Node       string       `yaml:"node"`
	CTID       int          `yaml:"ctid"`
	Image      string       `yaml:"image"`
	Tag        string       `yaml:"tag"`
	PullPolicy string       `yaml:"pullPolicy"`
	Resources  ResourceSpec `yaml:"resources"`
	Network    NetworkSpec  `yaml:"network"`
	Mounts     []MountSpec  `yaml:"mounts"`
	Health     HealthSpec   `yaml:"healthCheck"`
	Rollout    RolloutSpec  `yaml:"rollout"`
}

type ResourceSpec struct {
	Cores    int `yaml:"cores"`
	MemoryMB int `yaml:"memoryMB"`
}

type NetworkSpec struct {
	Bridge string `yaml:"bridge"`
	IP     string `yaml:"ip"`
	GW     string `yaml:"gw"`
}

type MountSpec struct {
	Host    string `yaml:"host"`
	Guest   string `yaml:"guest"`
	Options string `yaml:"options"`
}

type HealthSpec struct {
	Type             string `yaml:"type"`
	URL              string `yaml:"url"`
	TimeoutSeconds   int    `yaml:"timeoutSeconds"`
	IntervalSeconds  int    `yaml:"intervalSeconds"`
	HealthyThreshold int    `yaml:"healthyThreshold"`
}

type RolloutSpec struct {
	Strategy       string `yaml:"strategy"`
	MaxUnavailable int    `yaml:"maxUnavailable"`
	AutoRollback   bool   `yaml:"autoRollback"`
}

func ParseServiceSpec(data []byte) (ServiceSpec, error) {
	var svc ServiceSpec
	if err := yaml.Unmarshal(data, &svc); err != nil {
		return svc, fmt.Errorf("parse service spec: %w", err)
	}
	return svc, svc.Validate()
}

func LoadServiceSpecs(dir string) ([]ServiceSpec, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read services dir: %w", err)
	}
	var specs []ServiceSpec
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isSpecFile(entry) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read spec %s: %w", entry.Name(), err)
		}
		svc, err := ParseServiceSpec(bytes)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		specs = append(specs, svc)
	}
	return specs, nil
}

func isSpecFile(entry fs.DirEntry) bool {
	name := entry.Name()
	return filepath.Ext(name) == ".yml" || filepath.Ext(name) == ".yaml"
}

func (s ServiceSpec) Validate() error {
	if s.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if s.Spec.CTID <= 0 {
		return fmt.Errorf("spec.ctid must be > 0")
	}
	if s.Spec.Node == "" {
		return fmt.Errorf("spec.node is required")
	}
	if s.Spec.Image == "" {
		return fmt.Errorf("spec.image is required")
	}
	if s.Spec.Tag == "" {
		s.Spec.Tag = "latest"
	}
	if s.Spec.PullPolicy == "" {
		s.Spec.PullPolicy = "digest"
	}
	if s.Spec.Rollout.Strategy == "" {
		s.Spec.Rollout.Strategy = "recreate"
	}
	return nil
}
