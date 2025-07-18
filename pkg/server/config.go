package server

import "time"

// Config represents the complete server configuration
type Config struct {
	Server      ServerConfig      `mapstructure:"server" json:"server" yaml:"server"`
	Modules     ModulesConfig     `mapstructure:"modules" json:"modules" yaml:"modules"`
	DataSources DataSourcesConfig `mapstructure:"dataSources" json:"dataSources" yaml:"dataSources"`
	Ops         OpsConfig         `mapstructure:"ops" json:"ops" yaml:"ops"`
	SSE         SSEConfig         `mapstructure:"sse" json:"sse" yaml:"sse"`
	Auth        AuthConfig        `mapstructure:"auth" json:"auth" yaml:"auth"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port int    `mapstructure:"port" json:"port" yaml:"port"`
	Host string `mapstructure:"host" json:"host" yaml:"host"`
}

// ModulesConfig contains module-specific configurations
type ModulesConfig struct {
	Enabled   []string        `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Events    EventsConfig    `mapstructure:"events" json:"events" yaml:"events"`
	Metrics   MetricsConfig   `mapstructure:"metrics" json:"metrics" yaml:"metrics"`
	Logs      LogsConfig      `mapstructure:"logs" json:"logs" yaml:"logs"`
	Resources ResourcesConfig `mapstructure:"resources" json:"resources" yaml:"resources"`
	Alerts    AlertsConfig    `mapstructure:"alerts" json:"alerts" yaml:"alerts"`
}

// EventsConfig contains events module configuration
type EventsConfig struct {
	Endpoint     string        `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	PollInterval time.Duration `mapstructure:"pollInterval" json:"pollInterval" yaml:"pollInterval"`
}

// MetricsConfig contains Prometheus metrics module configuration
type MetricsConfig struct {
	Type                 string        `mapstructure:"type" json:"type" yaml:"type"`
	Endpoint             string        `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Timeout              time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	MaxConcurrentQueries int           `mapstructure:"maxConcurrentQueries" json:"maxConcurrentQueries" yaml:"maxConcurrentQueries"`
	QueryTimeout         time.Duration `mapstructure:"queryTimeout" json:"queryTimeout" yaml:"queryTimeout"`
	Retention            time.Duration `mapstructure:"retention" json:"retention" yaml:"retention"`
}

// LogsConfig contains Elasticsearch logs module configuration
type LogsConfig struct {
	Type          string        `mapstructure:"type" json:"type" yaml:"type"`
	Endpoint      string        `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Index         string        `mapstructure:"index" json:"index" yaml:"index"`
	Username      string        `mapstructure:"username" json:"username" yaml:"username"`
	Password      string        `mapstructure:"password" json:"password" yaml:"password"`
	MaxSize       int           `mapstructure:"maxSize" json:"maxSize" yaml:"maxSize"`
	Timeout       time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	ScrollTimeout time.Duration `mapstructure:"scrollTimeout" json:"scrollTimeout" yaml:"scrollTimeout"`
}

// ResourcesConfig contains resources module configuration
type ResourcesConfig struct {
	Endpoint string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	Type     string `mapstructure:"type" json:"type" yaml:"type"`
}

// AlertsConfig contains alerts module configuration
type AlertsConfig struct {
	Endpoint   string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	WebhookURL string `mapstructure:"webhookUrl" json:"webhookUrl" yaml:"webhookUrl"`
}

// DataSourcesConfig contains data source configurations
type DataSourcesConfig struct {
	Prometheus    PrometheusConfig    `mapstructure:"prometheus" json:"prometheus" yaml:"prometheus"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch" json:"elasticsearch" yaml:"elasticsearch"`
}

// PrometheusConfig contains Prometheus data source configuration
type PrometheusConfig struct {
	URL       string        `mapstructure:"url" json:"url" yaml:"url"`
	Timeout   time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	BasicAuth BasicAuth     `mapstructure:"basicAuth" json:"basicAuth" yaml:"basicAuth"`
}

// ElasticsearchConfig contains Elasticsearch data source configuration
type ElasticsearchConfig struct {
	URL   string            `mapstructure:"url" json:"url" yaml:"url"`
	Index string            `mapstructure:"index" json:"index" yaml:"index"`
	Auth  ElasticsearchAuth `mapstructure:"auth" json:"auth" yaml:"auth"`
	SSL   SSLConfig         `mapstructure:"ssl" json:"ssl" yaml:"ssl"`
}

// BasicAuth contains basic authentication configuration
type BasicAuth struct {
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

// ElasticsearchAuth contains Elasticsearch authentication configuration
type ElasticsearchAuth struct {
	Username string `mapstructure:"username" json:"username" yaml:"username"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

// SSLConfig contains SSL configuration
type SSLConfig struct {
	Enabled  bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	VerifyCA bool `mapstructure:"verifyCA" json:"verifyCA" yaml:"verifyCA"`
}

// OpsConfig contains ops service configuration
type OpsConfig struct {
	BaseURL string `mapstructure:"baseUrl" json:"baseUrl" yaml:"baseUrl"`
	APIKey  string `mapstructure:"apiKey" json:"apiKey" yaml:"apiKey"`
}

// SSEConfig contains Server-Sent Events configuration
type SSEConfig struct {
	KeepAlive      time.Duration `mapstructure:"keepAlive" json:"keepAlive" yaml:"keepAlive"`
	MaxConnections int           `mapstructure:"maxConnections" json:"maxConnections" yaml:"maxConnections"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled   bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	SecretKey string `mapstructure:"secretKey" json:"secretKey" yaml:"secretKey"`
}
