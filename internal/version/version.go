// Package version holds the build identity set by the linker (see Makefile
// and .goreleaser.yaml).
package version

// Set with -X github.com/kazysgurskas/argocd-hydrate/internal/version.<Name>=...
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)
