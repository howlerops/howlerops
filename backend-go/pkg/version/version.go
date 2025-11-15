package version

import (
	"fmt"
	"runtime"
)

// Version information - set at build time via ldflags
var (
	// Version is the semantic version (e.g., "2.0.0")
	Version = "dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// BuildDate is the build timestamp (RFC3339 format)
	BuildDate = "unknown"
)

// Info contains complete version information
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetInfo returns complete version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf(
		"Howlerops v%s\nCommit: %s\nBuilt: %s\nGo: %s\nOS/Arch: %s/%s",
		i.Version,
		i.Commit,
		i.BuildDate,
		i.GoVersion,
		i.OS,
		i.Arch,
	)
}

// ShortString returns a brief version string
func (i Info) ShortString() string {
	return fmt.Sprintf("v%s (%s)", i.Version, i.Commit[:min(7, len(i.Commit))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
