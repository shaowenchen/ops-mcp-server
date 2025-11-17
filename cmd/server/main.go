package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-mcp-server/cmd/version"
	"github.com/shaowenchen/ops-mcp-server/pkg/config"
	"github.com/shaowenchen/ops-mcp-server/pkg/docs"
	eventsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/events"
	logsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/logs"
	metricsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/metrics"
	sopsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/sops"
	tracesModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/traces"
)

// normalizeURI normalizes the URI path to ensure consistent handling of trailing slashes
func normalizeURI(uri string) string {
	if uri == "" {
		return "/mcp"
	}

	// Ensure URI starts with /
	if !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}

	// Remove trailing slash for consistency
	uri = strings.TrimSuffix(uri, "/")

	// Handle edge case where URI becomes empty after trimming
	if uri == "" {
		return "/mcp"
	}

	return uri
}

// parseEnabledModules parses the enabled query parameter and returns a map of enabled modules
func parseEnabledModules(queryParams string) map[string]bool {
	enabled := map[string]bool{
		"sops":    true, // default all enabled
		"events":  true,
		"metrics": true,
		"logs":    true,
		"traces":  true,
	}

	if queryParams == "" {
		return enabled
	}

	// Parse query parameters
	params := strings.Split(queryParams, "&")
	for _, param := range params {
		if strings.HasPrefix(param, "enabled=") {
			enabledStr := strings.TrimPrefix(param, "enabled=")
			if enabledStr == "" {
				continue
			}

			// Reset all to false first
			for k := range enabled {
				enabled[k] = false
			}

			// Enable only specified modules
			modules := strings.Split(enabledStr, ",")
			for _, module := range modules {
				module = strings.TrimSpace(module)
				if module != "" {
					enabled[module] = true
				}
			}
			break
		}
	}

	return enabled
}

// getEnabledModuleNames returns a slice of enabled module names
func getEnabledModuleNames(enabled map[string]bool) []string {
	var names []string
	for module, isEnabled := range enabled {
		if isEnabled {
			names = append(names, module)
		}
	}
	return names
}

var (
	cfgFile string
	logger  *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:     "ops-mcp-server",
	Short:   "Ops MCP Server - A modular operational monitoring server",
	Long:    `A modular MCP server providing operational monitoring capabilities including events, metrics, and logs.`,
	Run:     runServer,
	Version: version.Short(),
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build date, git commit, and platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.String())
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add version command
	rootCmd.AddCommand(versionCmd)

	// Configuration flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("host", "0.0.0.0", "Server host")
	rootCmd.PersistentFlags().Int("port", 80, "Server port")
	rootCmd.PersistentFlags().String("mode", "stdio", "Server mode: stdio or sse")
	rootCmd.PersistentFlags().String("uri", "/mcp", "MCP server URI path")

	// Module flags with different names to avoid conflicts
	// SOPS module
	rootCmd.PersistentFlags().Bool("enable-sops", false, "Enable SOPS module")

	// Events module
	rootCmd.PersistentFlags().Bool("enable-events", false, "Enable events module")

	// Metrics module
	rootCmd.PersistentFlags().Bool("enable-metrics", false, "Enable metrics module")

	// Logs module
	rootCmd.PersistentFlags().Bool("enable-logs", false, "Enable logs module")

	// Traces module
	rootCmd.PersistentFlags().Bool("enable-traces", false, "Enable traces module")

	// Bind flags to viper with unique keys
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("server.host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("server.mode", rootCmd.PersistentFlags().Lookup("mode"))
	viper.BindPFlag("server.uri", rootCmd.PersistentFlags().Lookup("uri"))

	// SOPS module bindings
	viper.BindPFlag("cli.sops.enabled", rootCmd.PersistentFlags().Lookup("enable-sops"))

	// Events module bindings
	viper.BindPFlag("cli.events.enabled", rootCmd.PersistentFlags().Lookup("enable-events"))

	// Metrics module bindings
	viper.BindPFlag("cli.metrics.enabled", rootCmd.PersistentFlags().Lookup("enable-metrics"))

	// Logs module bindings
	viper.BindPFlag("cli.logs.enabled", rootCmd.PersistentFlags().Lookup("enable-logs"))

	// Traces module bindings
	viper.BindPFlag("cli.traces.enabled", rootCmd.PersistentFlags().Lookup("enable-traces"))
}

func initConfig() {
	// Set environment variable key mapping for nested config
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set up specific environment variable mappings (only when env vars are set)
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.mode", "SERVER_MODE")
	viper.BindEnv("server.uri", "SERVER_URI")
	viper.BindEnv("server.token", "SERVER_TOKEN")
	viper.BindEnv("sops.ops.endpoint", "SOPS_OPS_ENDPOINT")
	viper.BindEnv("sops.ops.token", "SOPS_OPS_TOKEN")
	viper.BindEnv("events.ops.endpoint", "EVENTS_OPS_ENDPOINT")
	viper.BindEnv("events.ops.token", "EVENTS_OPS_TOKEN")
	viper.BindEnv("metrics.prometheus.endpoint", "METRICS_PROMETHEUS_ENDPOINT")
	viper.BindEnv("metrics.prometheus.username", "METRICS_PROMETHEUS_USERNAME")
	viper.BindEnv("metrics.prometheus.password", "METRICS_PROMETHEUS_PASSWORD")
	viper.BindEnv("metrics.prometheus.bearer_token", "METRICS_PROMETHEUS_BEARER_TOKEN")
	viper.BindEnv("logs.elasticsearch.endpoint", "LOGS_ELASTICSEARCH_ENDPOINT")
	viper.BindEnv("logs.elasticsearch.username", "LOGS_ELASTICSEARCH_USERNAME")
	viper.BindEnv("logs.elasticsearch.password", "LOGS_ELASTICSEARCH_PASSWORD")
	viper.BindEnv("logs.elasticsearch.api_key", "LOGS_ELASTICSEARCH_API_KEY")
	viper.BindEnv("traces.jaeger.endpoint", "TRACES_JAEGER_ENDPOINT")
	viper.BindEnv("traces.jaeger.timeout", "TRACES_JAEGER_TIMEOUT")
	// Module enablement environment variables
	viper.BindEnv("sops.enabled", "SOPS_ENABLED")
	viper.BindEnv("events.enabled", "EVENTS_ENABLED")
	viper.BindEnv("metrics.enabled", "METRICS_ENABLED")
	viper.BindEnv("logs.enabled", "LOGS_ENABLED")
	viper.BindEnv("traces.enabled", "TRACES_ENABLED")

	// Load main config file first
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Could not read config file: %v", err)
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	// Initialize logger
	var err error
	logLevel := viper.GetString("log.level")
	switch logLevel {
	case "debug":
		logger, err = zap.NewDevelopment()
	default:
		logger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	defer logger.Sync()

	// Get log level for debug logging
	logLevel := viper.GetString("log.level")

	// Load configuration
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		logger.Fatal("Failed to unmarshal config", zap.Error(err))
	}

	// Module enablement logic: CLI flags take precedence over environment variables
	// If CLI flag is set, use CLI value; otherwise use environment variable; otherwise use default (false)

	// Sops module
	if cmd.Flags().Changed("enable-sops") {
		// CLI flag takes precedence
		cfg.Sops.Enabled = viper.GetBool("cli.sops.enabled")
	} else {
		// Use environment variable or default to false
		cfg.Sops.Enabled = viper.GetBool("sops.enabled")
		if !viper.IsSet("sops.enabled") {
			cfg.Sops.Enabled = false // default
		}
	}

	// Events module
	if cmd.Flags().Changed("enable-events") {
		// CLI flag takes precedence
		cfg.Events.Enabled = viper.GetBool("cli.events.enabled")
	} else {
		// Use environment variable or default to false
		cfg.Events.Enabled = viper.GetBool("events.enabled")
		if !viper.IsSet("events.enabled") {
			cfg.Events.Enabled = false // default
		}
	}

	// Metrics module
	if cmd.Flags().Changed("enable-metrics") {
		// CLI flag takes precedence
		cfg.Metrics.Enabled = viper.GetBool("cli.metrics.enabled")
	} else {
		// Use environment variable or default to false
		cfg.Metrics.Enabled = viper.GetBool("metrics.enabled")
		if !viper.IsSet("metrics.enabled") {
			cfg.Metrics.Enabled = false // default
		}
	}

	// Logs module
	if cmd.Flags().Changed("enable-logs") {
		// CLI flag takes precedence
		cfg.Logs.Enabled = viper.GetBool("cli.logs.enabled")
	} else {
		// Use environment variable or default to false
		cfg.Logs.Enabled = viper.GetBool("logs.enabled")
		if !viper.IsSet("logs.enabled") {
			cfg.Logs.Enabled = false // default
		}
	}

	// Traces module
	if cmd.Flags().Changed("enable-traces") {
		// CLI flag takes precedence
		cfg.Traces.Enabled = viper.GetBool("cli.traces.enabled")
	} else {
		// Use environment variable or default to false
		cfg.Traces.Enabled = viper.GetBool("traces.enabled")
		if !viper.IsSet("traces.enabled") {
			cfg.Traces.Enabled = false // default
		}
	}

	// Get server mode - CLI flag takes precedence over config file
	serverMode := cfg.Server.Mode
	if viper.IsSet("server.mode") {
		serverMode = viper.GetString("server.mode")
	}
	if serverMode == "" {
		serverMode = "stdio" // default to stdio mode
	}

	logger.Info("Starting Ops MCP Server",
		zap.String("log_level", cfg.Log.Level),
		zap.String("mode", serverMode),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("uri", cfg.Server.URI),
		zap.Bool("sops_enabled", cfg.Sops.Enabled),
		zap.Bool("events_enabled", cfg.Events.Enabled),
		zap.Bool("metrics_enabled", cfg.Metrics.Enabled),
		zap.Bool("logs_enabled", cfg.Logs.Enabled),
		zap.Bool("traces_enabled", cfg.Traces.Enabled),
	)

	// Create MCP server
	mcpServer := server.NewMCPServer("ops-mcp-server", version.BuildVersion)

	// Register modules based on configuration
	var toolCount int
	var enabledTools []string
	var sopsTools []string
	var eventsTools []string
	var metricsTools []string
	var logsTools []string
	var tracesTools []string

	if cfg.Sops.Enabled {
		// Create Sops module instance with configuration
		sopsConfig := &sopsModule.Config{
			Tools: sopsModule.ToolsConfig{
				Prefix: cfg.Sops.Tools.Prefix,
				Suffix: cfg.Sops.Tools.Suffix,
			},
		}

		// Add Ops configuration if available
		if cfg.Sops.Ops != nil {
			sopsConfig.Endpoint = cfg.Sops.Ops.Endpoint
			sopsConfig.Token = cfg.Sops.Ops.Token
		}
		sopsModuleInstance, err := sopsModule.New(sopsConfig, logger)
		if err != nil {
			logger.Fatal("Failed to create SOPS module", zap.Error(err))
		}

		// Register tools
		sopsModuleTools := sopsModuleInstance.GetTools()
		for _, serverTool := range sopsModuleTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			enabledTools = append(enabledTools, serverTool.Tool.Name)
			sopsTools = append(sopsTools, serverTool.Tool.Name)
			toolCount++
		}

		logger.Info("SOPS module enabled", zap.Int("tools", len(sopsModuleTools)), zap.Strings("tool_names", sopsTools))
	}

	if cfg.Events.Enabled {
		// Create events module instance with configuration
		eventsConfig := &eventsModule.Config{
			PollInterval: 30 * time.Second, // default poll interval
			Tools: eventsModule.ToolsConfig{
				Prefix: cfg.Events.Tools.Prefix,
				Suffix: cfg.Events.Tools.Suffix,
			},
		}

		// Add Ops configuration if available
		if cfg.Events.Ops != nil {
			eventsConfig.Endpoint = cfg.Events.Ops.Endpoint
			eventsConfig.Token = cfg.Events.Ops.Token
		}
		eventsModuleInstance, err := eventsModule.New(eventsConfig, logger)
		if err != nil {
			logger.Fatal("Failed to create events module", zap.Error(err))
		}

		// Register tools
		eventsModuleTools := eventsModuleInstance.GetTools()
		for _, serverTool := range eventsModuleTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			enabledTools = append(enabledTools, serverTool.Tool.Name)
			eventsTools = append(eventsTools, serverTool.Tool.Name)
			toolCount++
		}

		logger.Info("Events module enabled", zap.Int("tools", len(eventsModuleTools)), zap.Strings("tool_names", eventsTools))
	}

	if cfg.Metrics.Enabled {
		// Create metrics module instance with configuration
		metricsConfig := &metricsModule.Config{
			Tools: metricsModule.ToolsConfig{
				Prefix: cfg.Metrics.Tools.Prefix,
				Suffix: cfg.Metrics.Tools.Suffix,
			},
		}

		// Add Prometheus configuration if available
		if cfg.Metrics.Prometheus != nil {
			metricsConfig.Prometheus = &metricsModule.PrometheusConfig{
				Endpoint: cfg.Metrics.Prometheus.Endpoint,
			}
		}

		metricsModuleInstance, err := metricsModule.New(metricsConfig, logger)
		if err != nil {
			logger.Fatal("Failed to create metrics module", zap.Error(err))
		}

		// Register tools
		metricsModuleTools := metricsModuleInstance.GetTools()
		for _, serverTool := range metricsModuleTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			enabledTools = append(enabledTools, serverTool.Tool.Name)
			metricsTools = append(metricsTools, serverTool.Tool.Name)
			toolCount++
		}

		logger.Info("Metrics module enabled", zap.Int("tools", len(metricsModuleTools)), zap.Strings("tool_names", metricsTools))
	}

	if cfg.Logs.Enabled {
		// Create logs module instance with configuration
		logsConfig := &logsModule.Config{
			Tools: logsModule.ToolsConfig{
				Prefix: cfg.Logs.Tools.Prefix,
				Suffix: cfg.Logs.Tools.Suffix,
			},
		}

		// Convert elasticsearch config if present
		if cfg.Logs.Elasticsearch != nil {
			logsConfig.Elasticsearch = &logsModule.ElasticsearchConfig{
				Endpoint: cfg.Logs.Elasticsearch.Endpoint,
				Username: cfg.Logs.Elasticsearch.Username,
				Password: cfg.Logs.Elasticsearch.Password,
				APIKey:   cfg.Logs.Elasticsearch.APIKey,
				Timeout:  cfg.Logs.Elasticsearch.Timeout,
			}
		}
		logsModuleInstance, err := logsModule.New(logsConfig, logger)
		if err != nil {
			logger.Fatal("Failed to create logs module", zap.Error(err))
		}

		// Register tools
		logsModuleTools := logsModuleInstance.GetTools()
		for _, serverTool := range logsModuleTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			enabledTools = append(enabledTools, serverTool.Tool.Name)
			logsTools = append(logsTools, serverTool.Tool.Name)
			toolCount++
		}

		logger.Info("Logs module enabled", zap.Int("tools", len(logsModuleTools)), zap.Strings("tool_names", logsTools))
	}

	if cfg.Traces.Enabled {
		// Create Jaeger module instance with configuration
		tracesConfig := &tracesModule.Config{
			Tools: tracesModule.ToolsConfig{
				Prefix: cfg.Traces.Tools.Prefix,
				Suffix: cfg.Traces.Tools.Suffix,
			},
		}

		// Add Jaeger configuration if available
		if cfg.Traces.Jaeger != nil {
			tracesConfig.Endpoint = cfg.Traces.Jaeger.Endpoint
			tracesConfig.Protocol = "HTTP" // default protocol
			tracesConfig.Port = 16686      // default port
			tracesConfig.Timeout = cfg.Traces.Jaeger.Timeout
		}
		tracesModuleInstance, err := tracesModule.New(tracesConfig, logger)
		if err != nil {
			logger.Fatal("Failed to create Jaeger module", zap.Error(err))
		}

		// Register tools
		tracesModuleTools := tracesModuleInstance.GetTools()
		for _, serverTool := range tracesModuleTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			enabledTools = append(enabledTools, serverTool.Tool.Name)
			tracesTools = append(tracesTools, serverTool.Tool.Name)
			toolCount++
		}

		logger.Info("Traces module enabled", zap.Int("tools", len(tracesModuleTools)), zap.Strings("tool_names", tracesTools))
	}

	if toolCount == 0 {
		logger.Warn("No modules enabled, server will have no tools available")
	} else {
		// Print detailed module and tool information
		logger.Info("=== Server Initialization Complete ===")
		logger.Info("Enabled modules and tools:")

		if cfg.Sops.Enabled {
			logger.Info("⚙️ Sops Module", zap.String("status", "enabled"), zap.Strings("tools", sopsTools))
		} else {
			logger.Info("⚙️ Sops Module", zap.String("status", "disabled"))
		}

		if cfg.Events.Enabled {
			logger.Info("📡 Events Module", zap.String("status", "enabled"), zap.Strings("tools", eventsTools))
		} else {
			logger.Info("📡 Events Module", zap.String("status", "disabled"))
		}

		if cfg.Metrics.Enabled {
			logger.Info("📊 Metrics Module", zap.String("status", "enabled"), zap.Strings("tools", metricsTools))
		} else {
			logger.Info("📊 Metrics Module", zap.String("status", "disabled"))
		}

		if cfg.Logs.Enabled {
			logger.Info("📋 Logs Module", zap.String("status", "enabled"), zap.Strings("tools", logsTools))
		} else {
			logger.Info("📋 Logs Module", zap.String("status", "disabled"))
		}

		if cfg.Traces.Enabled {
			logger.Info("🔍 Traces Module", zap.String("status", "enabled"), zap.Strings("tools", tracesTools))
		} else {
			logger.Info("🔍 Traces Module", zap.String("status", "disabled"))
		}

		logger.Info("All available tools:", zap.Strings("tools", enabledTools))
		logger.Info("Server initialized", zap.Int("total_tools", toolCount))
	}

	// Start server based on mode
	switch serverMode {
	case "stdio":
		logger.Info("Starting server in stdio mode")
		if err := server.ServeStdio(
			mcpServer,
		); err != nil {
			logger.Fatal("Stdio server failed", zap.Error(err))
		}
	case "sse":
		// Create a custom HTTP mux with health check endpoint
		mux := http.NewServeMux()

		// Get MCP URI from config and normalize it
		mcpURI := normalizeURI(cfg.Server.URI)

		// Add health check endpoint
		healthEndpoint := mcpURI + "/healthz"
		mux.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionInfo := version.Get()

			// Parse query parameters to show what modules would be enabled
			enabledModules := parseEnabledModules(r.URL.RawQuery)

			healthResponse := map[string]interface{}{
				"status":     "ok",
				"service":    "ops-mcp-server",
				"version":    versionInfo.Version,
				"build_date": versionInfo.BuildDate,
				"git_commit": versionInfo.GitCommit,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
				"mode":       serverMode,
				"endpoints": map[string]string{
					"mcp":     mcpURI,
					"sse":     mcpURI + "/sse",
					"message": mcpURI + "/message",
					"docs":    mcpURI + "/docs",
					"health":  healthEndpoint,
				},
				"modules": map[string]bool{
					"sops":    cfg.Sops.Enabled,
					"events":  cfg.Events.Enabled,
					"metrics": cfg.Metrics.Enabled,
					"logs":    cfg.Logs.Enabled,
					"traces":  cfg.Traces.Enabled,
				},
				"enabled_modules": enabledModules,
				"tools_count":     toolCount,
				"query_parameters": map[string]interface{}{
					"enabled": "sops,events,metrics,logs,traces (default: all enabled)",
					"example": mcpURI + "?enabled=sops,events",
				},
			}

			json.NewEncoder(w).Encode(healthResponse)
		})

		// Create custom HTTP server with optimized timeouts for MCP and TIME_WAIT management
		httpServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: mux,
			// Optimized timeouts for MCP server with TIME_WAIT reduction
			ReadTimeout:       30 * time.Second, // Reduce read timeout for faster connection release
			WriteTimeout:      30 * time.Second, // Reduce write timeout for faster connection release
			IdleTimeout:       60 * time.Second, // Reduce idle timeout for faster cleanup of idle connections
			ReadHeaderTimeout: 5 * time.Second,  // Quick header validation
		}

		// Create SSE server with dynamic base path
		sseServer := server.NewSSEServer(
			mcpServer,
			server.WithDynamicBasePath(func(r *http.Request, sessionID string) string {
				// Use the configured MCP URI as the base path
				return mcpURI
			}),
			server.WithBaseURL(fmt.Sprintf(":%d", cfg.Server.Port)),
			server.WithUseFullURLForMessageEndpoint(true),
		)

		// Add SSE and message endpoints using the SSE server handlers with debug logging
		sseEndpoint := mcpURI + "/sse"
		messageEndpoint := mcpURI + "/message"

		// Create wrapped handlers with debug logging
		sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logLevel == "debug" {
				logger.Debug("SSE connection request",
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.Strings("headers", getHeaderStrings(r.Header)),
				)
			}
			sseServer.SSEHandler().ServeHTTP(w, r)
		})

		messageHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logLevel == "debug" {
				logger.Debug("Message endpoint request",
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("content_type", r.Header.Get("Content-Type")),
					zap.String("content_length", r.Header.Get("Content-Length")),
					zap.Strings("headers", getHeaderStrings(r.Header)),
				)
			}
			sseServer.MessageHandler().ServeHTTP(w, r)
		})

		// Apply authentication middleware to SSE and message endpoints
		mux.Handle(sseEndpoint, authMiddleware(cfg.Server.Token)(sseHandler))
		mux.Handle(messageEndpoint, authMiddleware(cfg.Server.Token)(messageHandler))

		// Create a custom MCP handler that can parse query parameters
		mcpHandler := func(w http.ResponseWriter, r *http.Request) {
			// Log detailed request information in debug mode
			if logLevel == "debug" {
				logger.Debug("Incoming MCP request",
					zap.String("method", r.Method),
					zap.String("url", r.URL.String()),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("content_type", r.Header.Get("Content-Type")),
					zap.String("content_length", r.Header.Get("Content-Length")),
					zap.Strings("headers", getHeaderStrings(r.Header)),
				)
			}

			// Parse query parameters to determine enabled modules
			enabledModules := parseEnabledModules(r.URL.RawQuery)

			// Create a new MCP server instance for this request
			requestMCPServer := server.NewMCPServer("ops-mcp-server", version.BuildVersion)

			// Register modules based on query parameters
			var toolCount int
			var enabledTools []string

			if enabledModules["sops"] && cfg.Sops.Enabled {
				sopsConfig := &sopsModule.Config{
					Tools: sopsModule.ToolsConfig{
						Prefix: cfg.Sops.Tools.Prefix,
						Suffix: cfg.Sops.Tools.Suffix,
					},
				}
				if cfg.Sops.Ops != nil {
					sopsConfig.Endpoint = cfg.Sops.Ops.Endpoint
					sopsConfig.Token = cfg.Sops.Ops.Token
				}
				sopsModuleInstance, err := sopsModule.New(sopsConfig, logger)
				if err == nil {
					sopsModuleTools := sopsModuleInstance.GetTools()
					for _, serverTool := range sopsModuleTools {
						requestMCPServer.AddTool(serverTool.Tool, serverTool.Handler)
						enabledTools = append(enabledTools, serverTool.Tool.Name)
						toolCount++
					}
				}
			}

			if enabledModules["events"] && cfg.Events.Enabled {
				eventsConfig := &eventsModule.Config{
					PollInterval: 30 * time.Second,
					Tools: eventsModule.ToolsConfig{
						Prefix: cfg.Events.Tools.Prefix,
						Suffix: cfg.Events.Tools.Suffix,
					},
				}
				if cfg.Events.Ops != nil {
					eventsConfig.Endpoint = cfg.Events.Ops.Endpoint
					eventsConfig.Token = cfg.Events.Ops.Token
				}
				eventsModuleInstance, err := eventsModule.New(eventsConfig, logger)
				if err == nil {
					eventsModuleTools := eventsModuleInstance.GetTools()
					for _, serverTool := range eventsModuleTools {
						requestMCPServer.AddTool(serverTool.Tool, serverTool.Handler)
						enabledTools = append(enabledTools, serverTool.Tool.Name)
						toolCount++
					}
				}
			}

			if enabledModules["metrics"] && cfg.Metrics.Enabled {
				metricsConfig := &metricsModule.Config{
					Tools: metricsModule.ToolsConfig{
						Prefix: cfg.Metrics.Tools.Prefix,
						Suffix: cfg.Metrics.Tools.Suffix,
					},
				}
				if cfg.Metrics.Prometheus != nil {
					metricsConfig.Prometheus = &metricsModule.PrometheusConfig{
						Endpoint: cfg.Metrics.Prometheus.Endpoint,
					}
				}
				metricsModuleInstance, err := metricsModule.New(metricsConfig, logger)
				if err == nil {
					metricsModuleTools := metricsModuleInstance.GetTools()
					for _, serverTool := range metricsModuleTools {
						requestMCPServer.AddTool(serverTool.Tool, serverTool.Handler)
						enabledTools = append(enabledTools, serverTool.Tool.Name)
						toolCount++
					}
				}
			}

			if enabledModules["logs"] && cfg.Logs.Enabled {
				logsConfig := &logsModule.Config{
					Tools: logsModule.ToolsConfig{
						Prefix: cfg.Logs.Tools.Prefix,
						Suffix: cfg.Logs.Tools.Suffix,
					},
				}
				if cfg.Logs.Elasticsearch != nil {
					logsConfig.Elasticsearch = &logsModule.ElasticsearchConfig{
						Endpoint: cfg.Logs.Elasticsearch.Endpoint,
						Username: cfg.Logs.Elasticsearch.Username,
						Password: cfg.Logs.Elasticsearch.Password,
						APIKey:   cfg.Logs.Elasticsearch.APIKey,
						Timeout:  cfg.Logs.Elasticsearch.Timeout,
					}
				}
				logsModuleInstance, err := logsModule.New(logsConfig, logger)
				if err == nil {
					logsModuleTools := logsModuleInstance.GetTools()
					for _, serverTool := range logsModuleTools {
						requestMCPServer.AddTool(serverTool.Tool, serverTool.Handler)
						enabledTools = append(enabledTools, serverTool.Tool.Name)
						toolCount++
					}
				}
			}

			if enabledModules["traces"] && cfg.Traces.Enabled {
				tracesConfig := &tracesModule.Config{
					Tools: tracesModule.ToolsConfig{
						Prefix: cfg.Traces.Tools.Prefix,
						Suffix: cfg.Traces.Tools.Suffix,
					},
				}
				if cfg.Traces.Jaeger != nil {
					tracesConfig.Endpoint = cfg.Traces.Jaeger.Endpoint
					tracesConfig.Protocol = "HTTP"
					tracesConfig.Port = 16686
					tracesConfig.Timeout = cfg.Traces.Jaeger.Timeout
				}
				tracesModuleInstance, err := tracesModule.New(tracesConfig, logger)
				if err == nil {
					tracesModuleTools := tracesModuleInstance.GetTools()
					for _, serverTool := range tracesModuleTools {
						requestMCPServer.AddTool(serverTool.Tool, serverTool.Handler)
						enabledTools = append(enabledTools, serverTool.Tool.Name)
						toolCount++
					}
				}
			}

			// Log the request with enabled modules
			logger.Info("MCP request with enabled modules",
				zap.String("query", r.URL.RawQuery),
				zap.Strings("enabled_modules", getEnabledModuleNames(enabledModules)),
				zap.Int("tools_count", toolCount),
				zap.Strings("tools", enabledTools))

			// Create Streamable HTTP MCP server for this request
			streamableServer := server.NewStreamableHTTPServer(
				requestMCPServer,
				server.WithHeartbeatInterval(3*time.Second),
			)

			// Serve the request
			startTime := time.Now()
			streamableServer.ServeHTTP(w, r)

			// Log request completion in debug mode
			if logLevel == "debug" {
				duration := time.Since(startTime)
				logger.Debug("MCP request completed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Duration("duration", duration),
					zap.Strings("enabled_modules", getEnabledModuleNames(enabledModules)),
					zap.Int("tools_count", toolCount),
				)
			}
		}

		// Mount MCP handler to the mux with authentication middleware
		mux.Handle(mcpURI, authMiddleware(cfg.Server.Token)(http.HandlerFunc(mcpHandler)))

		// Add docs endpoint
		docsHandler := docs.NewHandler(&cfg, logger)
		mux.HandleFunc(mcpURI+"/docs", docsHandler.HandleDocs)

		// Start SSE server
		logger.Info("Starting server in SSE mode with health check",
			zap.String("address", httpServer.Addr),
			zap.String("health_endpoint", healthEndpoint),
			zap.String("mcp_endpoint", mcpURI),
			zap.String("sse_endpoint", sseEndpoint),
			zap.String("message_endpoint", messageEndpoint),
			zap.String("docs_endpoint", mcpURI+"/docs"))

		if err := httpServer.ListenAndServe(); err != nil {
			logger.Fatal("SSE server failed to start", zap.Error(err))
		}
	default:
		logger.Fatal("Invalid server mode", zap.String("mode", serverMode), zap.Strings("valid_modes", []string{"stdio", "sse"}))
	}
}

// getHeaderStrings converts http.Header to []string for logging
func getHeaderStrings(headers http.Header) []string {
	var headerStrings []string
	for name, values := range headers {
		for _, value := range values {
			headerStrings = append(headerStrings, fmt.Sprintf("%s: %s", name, value))
		}
	}
	return headerStrings
}

// authMiddleware creates an authentication middleware that validates the server token
func authMiddleware(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication if no token is configured
			if expectedToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header (Bearer token)
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token format
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization format. Expected 'Bearer <token>'", http.StatusUnauthorized)
				return
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				http.Error(w, "Token required", http.StatusUnauthorized)
				return
			}

			// Validate token
			if token != expectedToken {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Token is valid, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
