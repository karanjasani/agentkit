package main

import (
	"github.com/spf13/cobra"

	"github.com/karanjasani/agentkit/pkg/api"
)

func newOverviewCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "overview",
		Short: "Summarize the module: packages, entrypoints, generated/vendor folders",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Overview(cmd.Context())
			return emit(gf, res, err)
		},
	}
}

func newPackageCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "package <import-path|dir>",
		Short: "Show a package's imports, importers, exports and tests",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Package(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}

func newSymbolCmd(gf *globalFlags) *cobra.Command {
	var opts api.SymbolOptions
	cmd := &cobra.Command{
		Use:   "symbol <name>",
		Short: "Locate a symbol and return its signature, doc, body or shape",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Symbol(cmd.Context(), args[0], opts)
			return emit(gf, res, err)
		},
	}
	cmd.Flags().BoolVar(&opts.Body, "body", false, "return the full declaration source")
	cmd.Flags().BoolVar(&opts.SignatureOnly, "signature-only", false, "return only the signature")
	cmd.Flags().BoolVar(&opts.Doc, "doc", false, "return only the doc comment")
	cmd.Flags().BoolVar(&opts.Shape, "shape", false, "return the JSON contract (struct types)")
	return cmd
}

func newCallersCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "callers <name>",
		Short: "List direct and indirect callers of a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Callers(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}

func newDepsCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "deps <import-path|dir>",
		Short: "Show the intra-module dependency graph and depth",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Deps(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}

func newImpactCmd(gf *globalFlags) *cobra.Command {
	var base string
	cmd := &cobra.Command{
		Use:   "impact",
		Short: "Compute change impact of the working tree against a base ref",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Impact(cmd.Context(), api.ImpactOptions{Base: base})
			return emit(gf, res, err)
		},
	}
	cmd.Flags().StringVar(&base, "base", "HEAD", "git base ref to diff against")
	return cmd
}

func newTestsCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "tests <name>",
		Short: "List tests that exercise a symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Tests(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}

func newEndpointCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "endpoint <method> <path>",
		Short: "Trace a route through its handler to its upstream calls",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Endpoint(cmd.Context(), args[0], args[1])
			return emit(gf, res, err)
		},
	}
}

func newUpstreamsCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "upstreams <import-path|dir>",
		Short: "Map outbound REST/adapter calls under a package path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Upstreams(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}

func newStructCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "struct <name>",
		Short: "Show the recursive JSON contract of a struct type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newAnalyzer(cmd.Context(), gf)
			if err != nil {
				return emit(gf, nil, err)
			}
			res, err := a.Struct(cmd.Context(), args[0])
			return emit(gf, res, err)
		},
	}
}
