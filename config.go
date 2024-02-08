package itsy

import (
	"maps"
	"os"
	"slices"
	"strings"
)

const EnvPrefix = "ITSY_"

// Config defines a config structure that can be included in an application.
type Config struct {
	// URL is the NATS connection URL, e.g. "nats://user:pass@localhost:4222"
	URL string `yaml:"url" json:"url"`

	// Prefix is the service prefix, which defaults to "svc"
	Prefix string `yaml:"prefix" json:"prefix"`

	// Topologies is a list of dotted topologies to register the service under.
	// The services are registered under all parent levels, so there is no need
	// to explicitly add the parents of e.g. "eu.nl.ams".
	Topologies []string `yaml:"topologies" json:"topologies"`

	// Meta defines extra key-value string metadata that needs to be exposed in
	// the service metadata.
	Meta map[string]string `yaml:"meta" json:"meta"` // Extra metadata to expose in the service
}

func (c Config) AddEnviron() Config {
	if v := os.Getenv(EnvPrefix + "URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv(EnvPrefix + "PREFIX"); v != "" {
		c.Prefix = v
	}

	c.Topologies = slices.Clone(c.Topologies)
	if v := os.Getenv(EnvPrefix + "TOPO"); v != "" {
		for _, t := range strings.Fields(v) {
			c.Topologies = append(c.Topologies, t)
		}
	}

	c.Meta = maps.Clone(c.Meta)
	metaPrefix := EnvPrefix + "META_"
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, metaPrefix) {
			continue
		}
		k, v, ok := strings.Cut(env, "=")
		if !ok || v == "" {
			continue
		}
		metaKey := strings.ToLower(k[len(metaPrefix):])
		if metaKey == "" {
			continue
		}
		c.Meta[metaKey] = v
	}

	return c
}
