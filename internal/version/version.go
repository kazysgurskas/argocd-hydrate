// Package version holds the build identity set by the linker (see Makefile
// and .goreleaser.yaml).
package version

import "runtime/debug"

// Set with -X github.com/kazysgurskas/argocd-diff/internal/version.<Name>=...
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// A plain go build or go install has no linker flags, but Go records the
// module version and the VCS state in the binary; use those rather than
// reporting "dev".
func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch {
		case s.Key == "vcs.revision" && GitCommit == "none" && len(s.Value) >= 7:
			GitCommit = s.Value[:7]
		case s.Key == "vcs.time" && BuildDate == "unknown":
			BuildDate = s.Value
		}
	}
}
