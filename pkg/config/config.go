package config

import "time"

// Config represents the complete server configuration
type Config struct {
	Log     LogConfig     `mapstructure:"log" json:"log" yaml:"log"`
	Server  ServerConfig  `mapstructure:"server" json:"server" yaml:"server"`
	Events  EventsConfig  `mapstructure:"events" json:"events" yaml:"events"`
	Metrics MetricsConfig `mapstructure:"metrics" json:"metrics" yaml:"metrics"`
	Logs    LogsConfig    `mapstructure:"logs" json:"logs" yaml:"logs"`
	SSE     SSEConfig     `mapstructure:"sse" json:"sse" yaml:"sse"`
	Auth    AuthConfig    `mapstructure:"auth" json:"auth" yaml:"auth"`
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
}

// EventsConfig contains events module configuration
type EventsConfig struct {
	Enabled  bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Token    string `mapstructure:"token" json:"token" yaml:"token"`
}

// MetricsConfig contains metrics module configuration
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// LogsConfig contains logs module configuration
type LogsConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
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
