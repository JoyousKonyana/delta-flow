package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a deltaflow.yml file from path, parses it, and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("deltaflow.yml not found at %q — run deltaflow init to create one", path)
		}
		return nil, fmt.Errorf("failed to read config file at %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed deltaflow.yml at %q: %w", path, err)
	}

	return &cfg, nil
}
