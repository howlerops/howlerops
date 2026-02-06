package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion != runtime.Version() {
		t.Errorf("Expected Go version %s, got %s", runtime.Version(), info.GoVersion)
	}

	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info.OS)
	}

	if info.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, info.Arch)
	}
}

func TestInfo_String(t *testing.T) {
	info := Info{
		Version:   "2.0.0",
		Commit:    "abc123def456",
		BuildDate: "2024-10-23T10:00:00Z",
		GoVersion: "go1.21.0",
		OS:        "darwin",
		Arch:      "arm64",
	}

	str := info.String()

	// Check that all components are present
	if !strings.Contains(str, "Howlerops") {
		t.Error("String should contain 'Howlerops'")
	}

	if !strings.Contains(str, "2.0.0") {
		t.Error("String should contain version")
	}

	if !strings.Contains(str, "abc123def456") {
		t.Error("String should contain commit")
	}

	if !strings.Contains(str, "2024-10-23T10:00:00Z") {
		t.Error("String should contain build date")
	}

	if !strings.Contains(str, "go1.21.0") {
		t.Error("String should contain Go version")
	}

	if !strings.Contains(str, "darwin/arm64") {
		t.Error("String should contain OS/Arch")
	}
}

func TestInfo_ShortString(t *testing.T) {
	tests := []struct {
		name   string
		info   Info
		expect string
	}{
		{
			name: "full commit",
			info: Info{
				Version: "2.0.0",
				Commit:  "abc123def456",
			},
			expect: "v2.0.0 (abc123d)",
		},
		{
			name: "short commit",
			info: Info{
				Version: "1.0.0",
				Commit:  "abc",
			},
			expect: "v1.0.0 (abc)",
		},
		{
			name: "dev version",
			info: Info{
				Version: "dev",
				Commit:  "unknown",
			},
			expect: "vdev (unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.ShortString()
			if got != tt.expect {
				t.Errorf("Expected %q, got %q", tt.expect, got)
			}
		})
	}
}
