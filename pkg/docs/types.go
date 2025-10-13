package docs

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Module      string                 `json:"module"`
}

// ToolsInfoResponse represents the response structure for /mcp/docs
type ToolsInfoResponse struct {
	Service    string     `json:"service"`
	Version    string     `json:"version"`
	TotalTools int        `json:"total_tools"`
	Modules    []string   `json:"enabled_modules"`
	Tools      []ToolInfo `json:"tools"`
}
