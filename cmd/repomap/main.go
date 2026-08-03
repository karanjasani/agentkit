// Command repomap is a deterministic, read-only repository intelligence CLI for
// Go modules. It is a thin adapter over the pkg/api library: it parses flags,
// invokes the library, and renders the stable repomap.v1 output contract.
package main

import (
	"os"

	"github.com/karanjasani/agentkit/internal/repomap/output"
)

// lastExitCode carries the exit code set by a command's RunE. Commands write
// structured errors to stdout themselves (to preserve the JSON contract) and
// record the intended process exit code here.
var lastExitCode = output.ExitOK

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	lastExitCode = output.ExitOK
	root := newRootCmd()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		// Cobra-level errors are usage errors (bad flags, unknown command).
		return output.ExitUsageError
	}
	return lastExitCode
}
