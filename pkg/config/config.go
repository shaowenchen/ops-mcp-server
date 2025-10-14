package config

import "time"

// Config represents the complete server configuration
type Config struct {
	Log     LogConfig     `mapstructure:"log" json:"log" yaml:"log"`
	Server  ServerConfig  `mapstructure:"server" json:"server" yaml:"server"`
	Sops    SopsConfig    `mapstructure:"sops" json:"sops" yaml:"sops"`
	Events  EventsConfig  `mapstructure:"events" json:"events" yaml:"events"`
	Metrics MetricsConfig `mapstructure:"metrics" json:"metrics" yaml:"metrics"`
	Logs    LogsConfig    `mapstructure:"logs" json:"logs" yaml:"logs"`
	Traces  TracesConfig  `mapstructure:"traces" json:"traces" yaml:"traces"`
	SSE     SSEConfig     `mapstructure:"sse" json:"sse" yaml:"sse"`
	Auth    AuthConfig    `mapstructure:"auth" json:"auth" yaml:"auth"`
}

// ToolsConfig contains tools configuration
type ToolsConfig struct {
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Suffix string `mapstructure:"suffix" json:"suffix" yaml:"suffix"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level string `mapstructure:"level" json:"level" yaml:"level"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Host string `mapstructure:"host" json:"host" yaml:"host"`
	Port int    `mapstructure:"port" json:"port" yaml:"port"`
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`
	URI  string `mapstructure:"uri" json:"uri" yaml:"uri"`
}

// EventsOpsConfig contains Ops backend configuration for events
type EventsOpsConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Token    string `mapstructure:"token" json:"token" yaml:"token"`
}

// EventsConfig contains events module configuration
type EventsConfig struct {
	Enabled bool             `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Tools   ToolsConfig      `mapstructure:"tools" json:"tools" yaml:"tools"`
	Ops     *EventsOpsConfig `mapstructure:"ops" json:"ops" yaml:"ops"`
}

// PrometheusConfig contains Prometheus configuration for metrics
type PrometheusConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// MetricsConfig contains metrics module configuration
type MetricsConfig struct {
	Enabled    bool              `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Tools      ToolsConfig       `mapstructure:"tools" json:"tools" yaml:"tools"`
	Prometheus *PrometheusConfig `mapstructure:"prometheus" json:"prometheus" yaml:"prometheus"`
}

// LogsConfig contains logs module configuration
type LogsConfig struct {
	Enabled       bool                     `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Tools         ToolsConfig              `mapstructure:"tools" json:"tools" yaml:"tools"`
	Elasticsearch *LogsElasticsearchConfig `mapstructure:"elasticsearch" json:"elasticsearch" yaml:"elasticsearch"`
}

// LogsElasticsearchConfig contains elasticsearch backend configuration for logs
type LogsElasticsearchConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
	APIKey   string `mapstructure:"apikey" json:"apikey" yaml:"apikey"`
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// JaegerConfig contains Jaeger backend configuration for traces
type JaegerConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Timeout  int    `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
}

// TracesConfig contains traces module configuration
type TracesConfig struct {
	Enabled bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Tools   ToolsConfig   `mapstructure:"tools" json:"tools" yaml:"tools"`
	Jaeger  *JaegerConfig `mapstructure:"jaeger" json:"jaeger" yaml:"jaeger"`
}

// OpsConfig contains Ops backend configuration for Sops
type OpsConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Token    string `mapstructure:"token" json:"token" yaml:"token"`
}

// SopsConfig contains Sops module configuration
type SopsConfig struct {
	Enabled bool        `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Tools   ToolsConfig `mapstructure:"tools" json:"tools" yaml:"tools"`
	Ops     *OpsConfig  `mapstructure:"ops" json:"ops" yaml:"ops"`
}

// SSEConfig contains SSE configuration
type SSEConfig struct {
	KeepAlive      time.Duration `mapstructure:"keepAlive" json:"keepAlive" yaml:"keepAlive"`
	MaxConnections int           `mapstructure:"maxConnections" json:"maxConnections" yaml:"maxConnections"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}
