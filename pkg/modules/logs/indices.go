package logs

import "strings"

// normalizeCatFormat maps the format values advertised by the tools onto ones
// the _cat APIs accept. The tools document "table", but _cat only understands
// text/json/yaml/cbor/smile and rejects "table" outright. An unset format
// keeps the historical behaviour (structured JSON output), while "table" maps
// to the plain-text table ES actually renders.
func normalizeCatFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return "json"
	case "table", "text":
		return "text"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

// filterIndicesByStatus keeps only indices whose `status` column matches the
// requested open/close state. `status` is a _cat column rather than a filter
// parameter, and older versions (6.x) do not support expand_wildcards on
// _cat/indices at all, so the filtering is also applied locally to keep the
// tool's behaviour identical across versions.
func filterIndicesByStatus(indices []ElasticsearchIndex, status string) []ElasticsearchIndex {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return indices
	}

	filtered := make([]ElasticsearchIndex, 0, len(indices))
	for _, index := range indices {
		if strings.EqualFold(strings.TrimSpace(index.Status), status) {
			filtered = append(filtered, index)
		}
	}
	return filtered
}

// filterShardsByState keeps only shards whose state matches the requested
// value. _cat/shards has no state filter parameter; `s` only sorts.
func filterShardsByState(shards []ElasticsearchShard, state string) []ElasticsearchShard {
	state = strings.TrimSpace(state)
	if state == "" {
		return shards
	}

	filtered := make([]ElasticsearchShard, 0, len(shards))
	for _, shard := range shards {
		if strings.EqualFold(strings.TrimSpace(shard.State), state) {
			filtered = append(filtered, shard)
		}
	}
	return filtered
}
