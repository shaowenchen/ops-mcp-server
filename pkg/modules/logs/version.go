package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// versionRetryInterval limits how often we re-probe a cluster version when
// detection fails, so an unreachable backend does not add a round trip to
// every single tool call.
const versionRetryInterval = time.Minute

// esVersion is the major/minor part of the Elasticsearch cluster version.
type esVersion struct {
	Major int
	Minor int
	// Raw is the full version string reported by the cluster, e.g. "8.19.0".
	Raw string
}

// atLeast reports whether the version is greater than or equal to major.minor.
func (v esVersion) atLeast(major, minor int) bool {
	if v.Major != major {
		return v.Major > major
	}
	return v.Minor >= minor
}

// supportsESQL reports whether the cluster exposes the ES|QL _query endpoint.
// ES|QL was introduced as a technical preview in 8.11 and is generally
// available from 8.14; 7.x and earlier have no _query endpoint at all.
func (v esVersion) supportsESQL() bool {
	return v.atLeast(8, 11)
}

// supportsCatExpandWildcards reports whether _cat/indices accepts the
// expand_wildcards parameter. It was added in 7.0.
func (v esVersion) supportsCatExpandWildcards() bool {
	return v.atLeast(7, 0)
}

// parseESVersion parses a cluster version string such as "8.19.0",
// "7.17.9", or "8.11.0-SNAPSHOT".
func parseESVersion(raw string) (esVersion, error) {
	cleaned := strings.TrimSpace(raw)
	if i := strings.IndexAny(cleaned, "-+"); i >= 0 {
		cleaned = cleaned[:i]
	}
	parts := strings.Split(cleaned, ".")
	if len(parts) < 2 {
		return esVersion{}, fmt.Errorf("unexpected version format %q", raw)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return esVersion{}, fmt.Errorf("invalid major version in %q: %w", raw, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return esVersion{}, fmt.Errorf("invalid minor version in %q: %w", raw, err)
	}
	return esVersion{Major: major, Minor: minor, Raw: strings.TrimSpace(raw)}, nil
}

// versionCache stores the detected cluster version per endpoint. The SSE mode
// creates a fresh module per request, so the cache has to live at package
// level to avoid probing the cluster on every tool call.
type versionCache struct {
	mu        sync.Mutex
	versions  map[string]esVersion
	lastProbe map[string]time.Time
}

var clusterVersions = &versionCache{
	versions:  make(map[string]esVersion),
	lastProbe: make(map[string]time.Time),
}

// clusterVersion returns the cached cluster version, probing GET / when the
// cache is empty. The boolean result is false when the version is unknown
// (backend unreachable, unexpected response, ...). Callers must treat an
// unknown version as "assume modern" and simply try the request.
func (m *Module) clusterVersion(ctx context.Context) (esVersion, bool) {
	if m.config.Elasticsearch == nil || m.config.Elasticsearch.Endpoint == "" {
		return esVersion{}, false
	}
	endpoint := m.config.Elasticsearch.Endpoint

	clusterVersions.mu.Lock()
	if version, ok := clusterVersions.versions[endpoint]; ok {
		clusterVersions.mu.Unlock()
		return version, true
	}
	if last, ok := clusterVersions.lastProbe[endpoint]; ok && time.Since(last) < versionRetryInterval {
		clusterVersions.mu.Unlock()
		return esVersion{}, false
	}
	clusterVersions.mu.Unlock()

	detected, ok := m.detectClusterVersion(ctx)

	clusterVersions.mu.Lock()
	clusterVersions.lastProbe[endpoint] = time.Now()
	if ok {
		clusterVersions.versions[endpoint] = detected
	}
	clusterVersions.mu.Unlock()

	return detected, ok
}

// detectClusterVersion performs the actual GET / probe.
func (m *Module) detectClusterVersion(ctx context.Context) (esVersion, bool) {
	resp, err := m.makeElasticsearchRequest(ctx, "GET", "", nil)
	if err != nil {
		m.logger.Warn("Failed to detect Elasticsearch version", zap.Error(err))
		return esVersion{}, false
	}
	if resp == nil || resp.Body == nil {
		return esVersion{}, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Warn("Failed to read Elasticsearch version response", zap.Error(err))
		return esVersion{}, false
	}
	if resp.StatusCode >= 400 {
		m.logger.Warn("Elasticsearch version probe returned an error status",
			zap.Int("status_code", resp.StatusCode))
		return esVersion{}, false
	}

	var info struct {
		Version struct {
			Number string `json:"number"`
		} `json:"version"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		m.logger.Warn("Failed to parse Elasticsearch version response", zap.Error(err))
		return esVersion{}, false
	}

	version, err := parseESVersion(info.Version.Number)
	if err != nil {
		m.logger.Warn("Failed to parse Elasticsearch version",
			zap.String("version", info.Version.Number),
			zap.Error(err))
		return esVersion{}, false
	}

	m.logger.Info("Detected Elasticsearch cluster version",
		zap.String("version", version.Raw),
		zap.Bool("esql_supported", version.supportsESQL()))

	return version, true
}

// resetVersionCache drops cached versions. Tests use it to isolate cases.
func resetVersionCache() {
	clusterVersions.mu.Lock()
	clusterVersions.versions = make(map[string]esVersion)
	clusterVersions.lastProbe = make(map[string]time.Time)
	clusterVersions.mu.Unlock()
}
