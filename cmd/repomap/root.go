package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/karanjasani/agentkit/internal/repomap/output"
	"github.com/karanjasani/agentkit/internal/version"
	"github.com/karanjasani/agentkit/pkg/api"
)

// globalFlags holds flags shared by all subcommands.
type globalFlags struct {
	format string
	dir    string
}

func newRootCmd() *cobra.Command {
	gf := &globalFlags{}

	root := &cobra.Command{
		Use:   "repomap",
		Short: "Deterministic repository intelligence for AI coding agents",
		Long: "repomap analyzes a Go module and answers structural questions " +
			"(symbols, callers, dependencies, change impact, routes) as stable, " +
			"deterministic JSON. It is read-only and works entirely offline.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version(),
	}

	root.PersistentFlags().StringVar(&gf.format, "format", "json", "output format: json or text")
	root.PersistentFlags().StringVar(&gf.dir, "dir", ".", "module root directory to analyze")

	// Diagnostics go to stderr only; stdout is reserved for the JSON contract.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	root.AddCommand(
		newOverviewCmd(gf),
		newPackageCmd(gf),
		newSymbolCmd(gf),
		newCallersCmd(gf),
		newDepsCmd(gf),
		newImpactCmd(gf),
		newTestsCmd(gf),
		newEndpointCmd(gf),
		newUpstreamsCmd(gf),
		newStructCmd(gf),
	)
	return root
}

// emit renders either a successful result or a structured error and records the
// process exit code. It always returns nil so cobra does not additionally print.
func emit(gf *globalFlags, result any, err error) error {
	format, ferr := output.ParseFormat(gf.format)
	if ferr != nil {
		// Fall back to JSON for the error envelope.
		format = output.FormatJSON
		lastExitCode = output.Failure(os.Stdout, format, api.AsError(
			apiInvalid("--format: %v", ferr)))
		return nil
	}
	if err != nil {
		lastExitCode = output.Failure(os.Stdout, format, err)
		return nil
	}
	if werr := output.Success(os.Stdout, format, result); werr != nil {
		fmt.Fprintln(os.Stderr, "write error:", werr)
		lastExitCode = output.ExitError
	}
	return nil
}

// newAnalyzer constructs an Analyzer for the configured module directory.
func newAnalyzer(ctx context.Context, gf *globalFlags) (*api.Analyzer, error) {
	return api.New(ctx, api.WithDir(gf.dir))
}

// apiInvalid builds an INVALID_ARGUMENT error for CLI-level validation.
func apiInvalid(format string, args ...any) error {
	return &api.Error{Code: api.CodeInvalidArgument, Message: fmt.Sprintf(format, args...), Recoverable: true}
}
