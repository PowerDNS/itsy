package itsy

import (
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/PowerDNS/go-tlsconfig"
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

	// TLS is the optional TLS configuration to use for the NATS connection.
	// https://github.com/PowerDNS/go-tlsconfig
	TLS tlsconfig.Config `yaml:"tls" json:"tls"`

	// Username is the optional username to use for the NATS connection.
	Username string `yaml:"username" json:"username"`

	// Password is the optional password to use for the NATS connection.
	Password string `yaml:"password" json:"password"`

	// Token is the optional token to use for the NATS connection
	Token string `yaml:"token" json:"token"`
}

func (c Config) AddEnviron() Config {
	if v := os.Getenv(EnvPrefix + "URL"); v != "" {
		c.URL = v
	}
	if v := os.Getenv(EnvPrefix + "PREFIX"); v != "" {
		c.Prefix = v
	}

	if c.Topologies == nil {
		c.Topologies = []string{}
	} else {
		c.Topologies = slices.Clone(c.Topologies)
	}
	if v := os.Getenv(EnvPrefix + "TOPO"); v != "" {
		c.Topologies = append(c.Topologies, strings.Fields(v)...)
	}

	if c.Meta == nil {
		c.Meta = map[string]string{}
	} else {
		c.Meta = maps.Clone(c.Meta)
	}
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
