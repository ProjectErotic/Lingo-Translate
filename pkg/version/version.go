package version

import (
	"fmt"
	"os"
	"strings"
)

// Version is the current version of Lingo-Translate.
// It defaults to the hardcoded release version, but can be overridden at build-time via:
// -ldflags="-X 'nst-go/pkg/version.Version=2.3.0'"
var Version = "2.4.0"

func init() {
	// If VERSION file is found in working directory, use it as fallback
	if data, err := os.ReadFile("VERSION"); err == nil {
		v := strings.TrimSpace(string(data))
		if v != "" {
			Version = v
		}
	}
}

// Get returns the current version string.
func Get() string {
	return Version
}

// FullVersion returns the version with descriptive engine suffix.
func FullVersion() string {
	return fmt.Sprintf("%s (Go Pure Cross-Platform)", Version)
}
