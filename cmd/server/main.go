package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/mark3labs/mcp-go/server"
	"github.com/shaowenchen/ops-mcp-server/pkg/config"
	eventsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/events"
	logsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/logs"
	metricsModule "github.com/shaowenchen/ops-mcp-server/pkg/modules/metrics"
)

var (
	cfgFile string
	logger  *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:   "ops-mcp-server",
	Short: "Ops MCP Server - A modular operational monitoring server",
	Long:  `A modular MCP server providing operational monitoring capabilities including events, metrics, and logs.`,
	Run:   runServer,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Configuration flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/config.yaml)")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("host", "0.0.0.0", "Server host")
	rootCmd.PersistentFlags().Int("port", 3000, "Server port")
	rootCmd.PersistentFlags().String("mode", "stdio", "Server mode: stdio or sse")

	// Module flags with different names to avoid conflicts
	rootCmd.PersistentFlags().Bool("enable-events", true, "Enable events module")
	rootCmd.PersistentFlags().Bool("enable-metrics", false, "Enable metrics module")
	rootCmd.PersistentFlags().Bool("enable-logs", false, "Enable logs module")

	// Bind flags to viper with unique keys
	viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("server.host", rootCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("server.port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("server.mode", rootCmd.PersistentFlags().Lookup("mode"))
	viper.BindPFlag("cli.events.enabled", rootCmd.PersistentFlags().Lookup("enable-events"))
	viper.BindPFlag("cli.metrics.enabled", rootCmd.PersistentFlags().Lookup("enable-metrics"))
	viper.BindPFlag("cli.logs.enabled", rootCmd.PersistentFlags().Lookup("enable-logs"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("./configs")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Could not read config file: %v", err)
	}

	// Initialize logger
	var err error
	logLevel := viper.GetString("log_level")
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

	// Override module enablement with CLI flags if provided
	if viper.IsSet("cli.events.enabled") {
		cfg.Events.Enabled = viper.GetBool("cli.events.enabled")
	}
	if viper.IsSet("cli.metrics.enabled") {
		cfg.Metrics.Enabled = viper.GetBool("cli.metrics.enabled")
	}
	if viper.IsSet("cli.logs.enabled") {
		cfg.Logs.Enabled = viper.GetBool("cli.logs.enabled")
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
		zap.String("mode", serverMode),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.Bool("events_enabled", cfg.Events.Enabled),
		zap.Bool("metrics_enabled", cfg.Metrics.Enabled),
		zap.Bool("logs_enabled", cfg.Logs.Enabled),
	)

	// Create MCP server
	mcpServer := server.NewMCPServer("ops-mcp-server", "1.0.0")

	// Register modules based on configuration
	var toolCount int

	if cfg.Events.Enabled {
		eventsTools := eventsModule.GetTools()
		for _, serverTool := range eventsTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			toolCount++
		}
		logger.Info("Events module enabled", zap.Int("tools", len(eventsTools)))
	}

	if cfg.Metrics.Enabled {
		metricsTools := metricsModule.GetTools()
		for _, serverTool := range metricsTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			toolCount++
		}
		logger.Info("Metrics module enabled", zap.Int("tools", len(metricsTools)))
	}

	if cfg.Logs.Enabled {
		logsTools := logsModule.GetTools()
		for _, serverTool := range logsTools {
			mcpServer.AddTool(serverTool.Tool, serverTool.Handler)
			toolCount++
		}
		logger.Info("Logs module enabled", zap.Int("tools", len(logsTools)))
	}

	if toolCount == 0 {
		logger.Warn("No modules enabled, server will have no tools available")
	} else {
		logger.Info("Server initialized", zap.Int("total_tools", toolCount))
	}

	// Start server based on mode
	switch serverMode {
	case "stdio":
		logger.Info("Starting server in stdio mode")
		if err := server.ServeStdio(mcpServer); err != nil {
			logger.Fatal("Stdio server failed", zap.Error(err))
		}
	case "sse":
		// Create Streamable HTTP MCP server (SSE-based)
		streamableServer := server.NewStreamableHTTPServer(mcpServer)

		// Start SSE server
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("Starting server in SSE mode", zap.String("address", addr))

		if err := streamableServer.Start(addr); err != nil {
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
