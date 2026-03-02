package config

import "sort"

// ServiceConfig defines the watched paths for a single service.
type ServiceConfig struct {
	Paths []string `yaml:"paths"`
}

// Config is the top-level structure representing deltaflow.yml.
type Config struct {
	Version        int                      `yaml:"version"`
	Services       map[string]ServiceConfig `yaml:"services"`
	GlobalTriggers []string                 `yaml:"global_triggers"`
}

// ServiceNames returns a sorted slice of all service names defined in the config.
func (c *Config) ServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
