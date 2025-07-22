package logs

import "time"

// LogEntry represents a single log entry
type LogEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// Elasticsearch types for log storage backend

// ElasticsearchIndex represents an Elasticsearch index
type ElasticsearchIndex struct {
	Health      string `json:"health"`
	Status      string `json:"status"`
	Index       string `json:"index"`
	UUID        string `json:"uuid"`
	Primary     string `json:"pri"`
	Replica     string `json:"rep"`
	DocsCount   string `json:"docs.count"`
	DocsDeleted string `json:"docs.deleted"`
	StoreSize   string `json:"store.size"`
	PrimarySize string `json:"pri.store.size"`
}

// ElasticsearchMapping represents field mappings for an index
type ElasticsearchMapping struct {
	Index    string                 `json:"index"`
	Mappings map[string]interface{} `json:"mappings"`
	Settings map[string]interface{} `json:"settings"`
}

// ElasticsearchSearchHit represents a single search hit
type ElasticsearchSearchHit struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	ID     string                 `json:"_id"`
	Score  *float64               `json:"_score"`
	Source map[string]interface{} `json:"_source"`
	Fields map[string]interface{} `json:"fields,omitempty"`
	Sort   []interface{}          `json:"sort,omitempty"`
}

// ElasticsearchSearchHits represents search hits collection
type ElasticsearchSearchHits struct {
	Total    ElasticsearchTotal       `json:"total"`
	MaxScore *float64                 `json:"max_score"`
	Hits     []ElasticsearchSearchHit `json:"hits"`
}

// ElasticsearchTotal represents total hits info
type ElasticsearchTotal struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

// ElasticsearchSearchResponse represents search response
type ElasticsearchSearchResponse struct {
	Took         int64                   `json:"took"`
	TimedOut     bool                    `json:"timed_out"`
	Shards       ElasticsearchShards     `json:"_shards"`
	Hits         ElasticsearchSearchHits `json:"hits"`
	Aggregations map[string]interface{}  `json:"aggregations,omitempty"`
}

// ElasticsearchShards represents shards info
type ElasticsearchShards struct {
	Total      int64 `json:"total"`
	Successful int64 `json:"successful"`
	Skipped    int64 `json:"skipped"`
	Failed     int64 `json:"failed"`
}

// ElasticsearchShard represents a single shard
type ElasticsearchShard struct {
	Index            string `json:"index"`
	Shard            string `json:"shard"`
	Prirep           string `json:"prirep"`
	State            string `json:"state"`
	Docs             string `json:"docs"`
	Store            string `json:"store"`
	IP               string `json:"ip"`
	Node             string `json:"node"`
	UnassignedReason string `json:"unassigned.reason,omitempty"`
}

// ElasticsearchClusterHealth represents cluster health
type ElasticsearchClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int64   `json:"number_of_nodes"`
	NumberOfDataNodes           int64   `json:"number_of_data_nodes"`
	ActivePrimaryShards         int64   `json:"active_primary_shards"`
	ActiveShards                int64   `json:"active_shards"`
	RelocatingShards            int64   `json:"relocating_shards"`
	InitializingShards          int64   `json:"initializing_shards"`
	UnassignedShards            int64   `json:"unassigned_shards"`
	DelayedUnassignedShards     int64   `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int64   `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int64   `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int64   `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// ESQLResponse represents ES|QL query response
type ESQLResponse struct {
	Columns []ESQLColumn           `json:"columns"`
	Values  [][]interface{}        `json:"values"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// ESQLColumn represents ES|QL column definition
type ESQLColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ElasticsearchRequest represents a generic ES request
type ElasticsearchRequest struct {
	Index   string                 `json:"index,omitempty"`
	Query   map[string]interface{} `json:"query,omitempty"`
	Size    *int                   `json:"size,omitempty"`
	From    *int                   `json:"from,omitempty"`
	Sort    []interface{}          `json:"sort,omitempty"`
	Source  interface{}            `json:"_source,omitempty"`
	Timeout string                 `json:"timeout,omitempty"`
}

// ElasticsearchError represents an ES error response
type ElasticsearchError struct {
	Type         string               `json:"type"`
	Reason       string               `json:"reason"`
	Index        string               `json:"index,omitempty"`
	ResourceType string               `json:"resource.type,omitempty"`
	ResourceID   string               `json:"resource.id,omitempty"`
	RootCause    []ElasticsearchError `json:"root_cause,omitempty"`
	CausedBy     *ElasticsearchError  `json:"caused_by,omitempty"`
}

// ElasticsearchAPIError represents API error response
type ElasticsearchAPIError struct {
	Error  ElasticsearchError `json:"error"`
	Status int                `json:"status"`
}
