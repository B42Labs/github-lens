package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHubToken   string      `yaml:"github_token"`
	Organizations []OrgConfig `yaml:"organizations"`
	Server        ServerConfig `yaml:"server"`
	Sync          SyncConfig   `yaml:"sync"`
}

type OrgConfig struct {
	Name         string   `yaml:"name"`
	IncludeRepos []string `yaml:"include_repos"`
	ExcludeRepos []string `yaml:"exclude_repos"`
}

type ServerConfig struct {
	Port       int    `yaml:"port"`
	CORSOrigin string `yaml:"cors_origin"`
}

type SyncConfig struct {
	Interval    string `yaml:"interval"`
	Concurrency int    `yaml:"concurrency"`
}

func (c *Config) SyncInterval() time.Duration {
	if c.Sync.Interval == "" || c.Sync.Interval == "0" {
		return 0
	}
	d, err := time.ParseDuration(c.Sync.Interval)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}

func Load(path string) (*Config, error) {
	var data []byte
	var err error

	if path != "" {
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config %s: %w", path, err)
		}
	} else {
		// Search CWD first, then ~/.config/github-lens/
		candidates := []string{"config.yaml"}
		home, herr := os.UserHomeDir()
		if herr == nil {
			candidates = append(candidates, filepath.Join(home, ".config", "github-lens", "config.yaml"))
		}
		for _, c := range candidates {
			data, err = os.ReadFile(c)
			if err == nil {
				break
			}
		}
		if data == nil {
			return nil, fmt.Errorf("no config.yaml found in current directory or ~/.config/github-lens/")
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Sync.Interval == "" {
		cfg.Sync.Interval = "15m"
	}
	if cfg.Sync.Concurrency == 0 {
		cfg.Sync.Concurrency = 5
	}

	// Env var override
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cfg.GitHubToken = token
	}

	// Validation
	if len(cfg.Organizations) == 0 {
		return nil, fmt.Errorf("at least one organization must be configured")
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("server port must be between 1 and 65535")
	}
	if cfg.Sync.Concurrency < 1 {
		return nil, fmt.Errorf("sync concurrency must be at least 1")
	}

	return &cfg, nil
}
