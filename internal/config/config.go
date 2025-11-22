package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
	"os"
)

type RegistryConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type PVEConfig struct {
	Mode       string `yaml:"mode"`
	PctPath    string `yaml:"pctPath"`
	Node       string `yaml:"node"`
	StatePath  string `yaml:"statePath"`
	DryRun     bool   `yaml:"dryRun"`
	APIToken   string `yaml:"apiToken"`
	APIURL     string `yaml:"apiUrl"`
	APITokenID string `yaml:"apiTokenId"`
}

type RunnerConfig struct {
	ServicesPath string        `yaml:"servicesPath"`
	Interval     time.Duration `yaml:"interval"`
}

type Config struct {
	Registry RegistryConfig `yaml:"registry"`
	PVE      PVEConfig      `yaml:"pve"`
	Runner   RunnerConfig   `yaml:"runner"`
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.Runner.ServicesPath == "" {
		return fmt.Errorf("runner.servicesPath is required")
	}
	if c.Runner.Interval == 0 {
		c.Runner.Interval = 10 * time.Second
	}
	switch c.PVE.Mode {
	case "cli", "api", "":
		if c.PVE.Mode == "" {
			c.PVE.Mode = "cli"
		}
	default:
		return fmt.Errorf("unknown pve.mode %q", c.PVE.Mode)
	}
	if c.PVE.Mode == "cli" && c.PVE.PctPath == "" {
		c.PVE.PctPath = "pct"
	}
	return nil
}
