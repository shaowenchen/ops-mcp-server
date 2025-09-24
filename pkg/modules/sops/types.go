package sops

import (
	v1 "github.com/shaowenchen/ops/api/v1"
)

// SOPSConfig represents a standard operation procedure
type SOPSConfig struct {
	Desc      string       `json:"desc,omitempty" yaml:"desc,omitempty"`
	Variables v1.Variables `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// Config contains sops module configuration
type Config struct {
	Endpoint string      `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Token    string      `mapstructure:"token" json:"token" yaml:"token"`
	Tools    ToolsConfig `mapstructure:"tools" json:"tools" yaml:"tools"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}
