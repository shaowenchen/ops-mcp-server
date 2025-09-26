
package version

import (
	"fmt"
	"runtime"
)

// These variables are set during build time via ldflags
var (
	BuildVersion = "latest"
	BuildDate    = "unknown"
	GitCommitID  = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   BuildVersion,
		BuildDate: BuildDate,
		GitCommit: GitCommitID,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func String() string {
	info := Get()
	return fmt.Sprintf("ops-mcp-server %s (built on %s, commit %s, %s %s)",
		info.Version, info.BuildDate, info.GitCommit, info.GoVersion, info.Platform)
}

// Short returns a short version string with build date and commit
func Short() string {
	return fmt.Sprintf("%s (built on %s, commit %s)",
		BuildVersion, BuildDate, GitCommitID)
}
