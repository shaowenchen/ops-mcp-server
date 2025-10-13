package docs

import (
	"encoding/json"
	"net/http"

	"github.com/shaowenchen/ops-mcp-server/pkg/config"
	"go.uber.org/zap"
)

// Handler handles documentation requests
type Handler struct {
	collector *Collector
	logger    *zap.Logger
}

// NewHandler creates a new docs handler
func NewHandler(cfg *config.Config, logger *zap.Logger) *Handler {
	return &Handler{
		collector: NewCollector(cfg, logger),
		logger:    logger,
	}
}

// HandleDocs handles the /mcp/docs endpoint
func (h *Handler) HandleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Collect tool information from all enabled modules
	toolsInfo := h.collector.CollectToolsInfo()

	if err := json.NewEncoder(w).Encode(toolsInfo); err != nil {
		h.logger.Error("Failed to encode tools info", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
