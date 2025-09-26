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
	eventsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/events"
	logsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/logs"
	metricsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/metrics"
	sopsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/sops"
	tracesModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/traces"
)

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
	// Enable automatic environment variable support first
	viper.AutomaticEnv()

	// Set environment variable key mapping for nested config
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set up specific environment variable mappings
	viper.BindEnv("log.level", "LOG_LEVEL")
	viper.BindEnv("server.host", "SERVER_HOST")
	viper.BindEnv("server.port", "SERVER_PORT")
	viper.BindEnv("server.mode", "SERVER_MODE")
	viper.BindEnv("sops.ops.endpoint", "SOPS_OPS_ENDPOINT")
	viper.BindEnv("sops.ops.token", "SOPS_OPS_TOKEN")
	viper.BindEnv("events.ops.endpoint", "EVENTS_OPS_ENDPOINT")
	viper.BindEnv("events.ops.token", "EVENTS_OPS_TOKEN")
	viper.BindEnv("metrics.prometheus.endpoint", "METRICS_PROMETHEUS_ENDPOINT")
	viper.BindEnv("logs.elasticsearch.endpoint", "LOGS_ELASTICSEARCH_ENDPOINT")
	viper.BindEnv("logs.elasticsearch.username", "LOGS_ELASTICSEARCH_USERNAME")
	viper.BindEnv("logs.elasticsearch.password", "LOGS_ELASTICSEARCH_PASSWORD")
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
			Endpoint: cfg.Sops.Ops.Endpoint,
			Token:    cfg.Sops.Ops.Token,
			Tools: sopsModule.ToolsConfig{
				Prefix: cfg.Sops.Tools.Prefix,
				Suffix: cfg.Sops.Tools.Suffix,
			},
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
			Endpoint:     cfg.Events.Ops.Endpoint,
			Token:        cfg.Events.Ops.Token,
			PollInterval: 30 * time.Second, // default poll interval
			Tools: eventsModule.ToolsConfig{
				Prefix: cfg.Events.Tools.Prefix,
				Suffix: cfg.Events.Tools.Suffix,
			},
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

		// Add health check endpoint
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionInfo := version.Get()
			healthResponse := map[string]interface{}{
				"status":     "ok",
				"service":    "ops-mcp-server",
				"version":    versionInfo.Version,
				"build_date": versionInfo.BuildDate,
				"git_commit": versionInfo.GitCommit,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
				"mode":       serverMode,
				"modules": map[string]bool{
					"sops":    cfg.Sops.Enabled,
					"events":  cfg.Events.Enabled,
					"metrics": cfg.Metrics.Enabled,
					"logs":    cfg.Logs.Enabled,
					"traces":  cfg.Traces.Enabled,
				},
				"tools_count": toolCount,
			}

			json.NewEncoder(w).Encode(healthResponse)
		})

		// Create custom HTTP server
		httpServer := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler: mux,
		}

		// Create Streamable HTTP MCP server with custom server
		streamableServer := server.NewStreamableHTTPServer(
			mcpServer,
			server.WithStreamableHTTPServer(httpServer),
			server.WithEndpointPath("/mcp"),
			server.WithHeartbeatInterval(3*time.Second),
		)

		// Mount MCP handler to the mux
		mux.Handle("/mcp", streamableServer)

		// Start SSE server
		logger.Info("Starting server in SSE mode with health check",
			zap.String("address", httpServer.Addr),
			zap.String("health_endpoint", "/healthz"),
			zap.String("mcp_endpoint", "/mcp"))

		if err := streamableServer.Start(httpServer.Addr); err != nil {
			logger.Fatal("SSE server failed to start", zap.Error(err))
		}
	default:
		logger.Fatal("Invalid server mode", zap.String("mode", serverMode), zap.Strings("valid_modes", []string{"stdio", "sse"}))
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
