// Package buildinfo exposes metadata injected into release binaries by the
// build scripts. The defaults identify binaries created directly with go run
// or go build.
package buildinfo

import "fmt"

var (
	Version = "dev"
	Build   = "0"
	Commit  = "unknown"
)

func Summary() string {
	return fmt.Sprintf("%s (build %s, commit %s)", Version, Build, Commit)
}
