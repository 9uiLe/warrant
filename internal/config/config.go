package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SpecRoot           string   `yaml:"spec_root"`
	TestGlobs          []string `yaml:"test_globs"`
	Tag                string   `yaml:"tag"`
	IDPattern          string   `yaml:"id_pattern"`
	DerivedGlobs       []string `yaml:"derived_globs"`
	ReportPath         string   `yaml:"report_path"`
	EnforceTag         string   `yaml:"enforce_tag"`
	SemanticCommand    string   `yaml:"semantic_command"`
	SemanticTimeoutSec int      `yaml:"semantic_timeout_sec"`
}

// Load はファイルを読み込み、無いキーだけ既定で埋める（setdefault 相当）
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	applyDefaults(&c)
	return &c, nil
}

func Default() *Config {
	c := &Config{}
	applyDefaults(c)
	return c
}

func applyDefaults(c *Config) {
	if c.SpecRoot == "" {
		c.SpecRoot = "docs/spec"
	}
	if len(c.TestGlobs) == 0 {
		c.TestGlobs = []string{"**/*_test.*"}
	}
	if c.Tag == "" {
		c.Tag = "@covers"
	}
	if c.IDPattern == "" {
		c.IDPattern = `[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+`
	}
	// DerivedGlobs: nil スライスは既定で空（[]string{}）として扱う
	if c.DerivedGlobs == nil {
		c.DerivedGlobs = []string{}
	}
	if c.ReportPath == "" {
		c.ReportPath = ".warrant/traceability.generated.md"
	}
	if c.EnforceTag == "" {
		c.EnforceTag = "@warrant-enforces"
	}
	if c.SemanticTimeoutSec == 0 {
		c.SemanticTimeoutSec = 30
	}
}
