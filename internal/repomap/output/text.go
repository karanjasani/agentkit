package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/karanjasani/agentkit/pkg/models"
)

// renderText writes a compact human-readable representation of a result. It is
// intentionally best-effort: JSON is the canonical, complete format.
func renderText(w io.Writer, result any) error {
	switch r := result.(type) {
	case models.Overview:
		return renderOverview(w, r)
	case models.Package:
		return renderPackage(w, r)
	case models.Symbol:
		return renderSymbol(w, r)
	case models.Callers:
		return renderCallers(w, r)
	case models.Deps:
		return renderDeps(w, r)
	case models.Impact:
		return renderImpact(w, r)
	case models.Tests:
		return renderTests(w, r)
	case models.Endpoint:
		return renderEndpoint(w, r)
	case models.Upstreams:
		return renderUpstreams(w, r)
	case models.Struct:
		return renderStruct(w, r, 0)
	default:
		_, err := fmt.Fprintf(w, "%+v\n", result)
		return err
	}
}

func loc(l models.Location) string {
	if l.File == "" {
		return "?"
	}
	return fmt.Sprintf("%s:%d", l.File, l.Line)
}

func renderOverview(w io.Writer, o models.Overview) error {
	fmt.Fprintf(w, "module: %s\n", o.Module)
	if o.GoVersion != "" {
		fmt.Fprintf(w, "go: %s\n", o.GoVersion)
	}
	fmt.Fprintf(w, "packages: %d  files: %d  entrypoints: %d\n",
		o.Stats.Packages, o.Stats.Files, o.Stats.Entrypoints)
	if len(o.Entrypoints) > 0 {
		fmt.Fprintln(w, "\nentrypoints:")
		for _, p := range o.Entrypoints {
			fmt.Fprintf(w, "  %s (%s)\n", p.ImportPath, p.Dir)
		}
	}
	fmt.Fprintln(w, "\npackages:")
	for _, p := range o.Packages {
		fmt.Fprintf(w, "  %s\n", p.ImportPath)
	}
	return nil
}

func renderPackage(w io.Writer, p models.Package) error {
	fmt.Fprintf(w, "package %s (%s)\n", p.Name, p.ImportPath)
	fmt.Fprintf(w, "dir: %s\n", p.Dir)
	fmt.Fprintf(w, "\nimports (%d):\n", len(p.Imports))
	for _, i := range p.Imports {
		fmt.Fprintf(w, "  %s\n", i)
	}
	fmt.Fprintf(w, "\nimported by (%d):\n", len(p.ImportedBy))
	for _, i := range p.ImportedBy {
		fmt.Fprintf(w, "  %s\n", i)
	}
	fmt.Fprintf(w, "\nexports (%d):\n", len(p.Exports))
	for _, s := range p.Exports {
		fmt.Fprintf(w, "  %s %s  %s\n", s.Kind, s.Name, loc(s.Location))
	}
	if len(p.TestFiles) > 0 {
		fmt.Fprintf(w, "\ntest files (%d):\n", len(p.TestFiles))
		for _, f := range p.TestFiles {
			fmt.Fprintf(w, "  %s\n", f)
		}
	}
	return nil
}

func renderSymbol(w io.Writer, s models.Symbol) error {
	fmt.Fprintf(w, "%s %s\n", s.Kind, s.Name)
	fmt.Fprintf(w, "package: %s\n", s.Package)
	fmt.Fprintf(w, "location: %s\n", loc(s.Location))
	if s.Signature != "" {
		fmt.Fprintf(w, "signature: %s\n", s.Signature)
	}
	if s.Doc != "" {
		fmt.Fprintf(w, "\ndoc:\n%s\n", s.Doc)
	}
	if s.Body != "" {
		fmt.Fprintf(w, "\nbody:\n%s\n", s.Body)
	}
	if s.Shape != nil {
		fmt.Fprintln(w, "\nshape:")
		_ = renderStruct(w, *s.Shape, 1)
	}
	if len(s.Callers) > 0 {
		fmt.Fprintf(w, "\ncallers (%d):\n", len(s.Callers))
		for _, c := range s.Callers {
			fmt.Fprintf(w, "  %s  %s  [%s]\n", loc(c.Location), strings.TrimSpace(c.Context), c.Confidence)
		}
	}
	return nil
}

func renderCallers(w io.Writer, c models.Callers) error {
	fmt.Fprintf(w, "callers of %s (%s)\n", c.Symbol, c.Package)
	fmt.Fprintf(w, "\ndirect (%d):\n", len(c.Direct))
	for _, d := range c.Direct {
		fmt.Fprintf(w, "  %s  %s  %s  [%s]\n", d.Symbol, loc(d.Location), strings.TrimSpace(d.Context), d.Confidence)
	}
	fmt.Fprintf(w, "\nindirect (%d):\n", len(c.Indirect))
	for _, d := range c.Indirect {
		fmt.Fprintf(w, "  %s  %s  [%s]\n", d.Symbol, loc(d.Location), d.Confidence)
	}
	return nil
}

func renderDeps(w io.Writer, d models.Deps) error {
	fmt.Fprintf(w, "deps of %s (depth %d)\n", d.Root, d.Depth)
	fmt.Fprintf(w, "\nnodes (%d):\n", len(d.Nodes))
	for _, n := range d.Nodes {
		fmt.Fprintf(w, "  %s\n", n)
	}
	fmt.Fprintf(w, "\nedges (%d):\n", len(d.Edges))
	for _, e := range d.Edges {
		fmt.Fprintf(w, "  %s -> %s\n", e.From, e.To)
	}
	return nil
}

func renderImpact(w io.Writer, i models.Impact) error {
	fmt.Fprintf(w, "impact vs %s\n", i.Base)
	fmt.Fprintf(w, "risk: %d (%s)\n", i.RiskScore, i.RiskLevel)
	fmt.Fprintf(w, "\nchanged packages (%d):\n", len(i.ChangedPackages))
	for _, p := range i.ChangedPackages {
		fmt.Fprintf(w, "  %s\n", p)
	}
	fmt.Fprintf(w, "\naffected packages (%d):\n", len(i.AffectedPackages))
	for _, p := range i.AffectedPackages {
		fmt.Fprintf(w, "  %s\n", p)
	}
	fmt.Fprintf(w, "\nrecommended tests (%d):\n", len(i.RecommendedTests))
	for _, t := range i.RecommendedTests {
		fmt.Fprintf(w, "  %s\n", t)
	}
	return nil
}

func renderTests(w io.Writer, t models.Tests) error {
	fmt.Fprintf(w, "tests for %s\n", t.Symbol)
	writeTestGroup(w, "unit", t.Unit)
	writeTestGroup(w, "integration", t.Integration)
	writeTestGroup(w, "benchmark", t.Benchmark)
	return nil
}

func writeTestGroup(w io.Writer, label string, tests []models.Test) {
	fmt.Fprintf(w, "\n%s (%d):\n", label, len(tests))
	for _, t := range tests {
		fmt.Fprintf(w, "  %s  %s\n", t.Name, loc(t.Location))
	}
}

func renderEndpoint(w io.Writer, e models.Endpoint) error {
	fmt.Fprintf(w, "%s %s  [%s, %s]\n", e.Method, e.Path, e.Framework, e.Confidence)
	fmt.Fprintf(w, "route: %s\n", loc(e.Route))
	if e.Handler != nil {
		fmt.Fprintf(w, "handler: %s  %s\n", e.Handler.Name, loc(e.Handler.Location))
	}
	if e.RequestType != "" {
		fmt.Fprintf(w, "request: %s\n", e.RequestType)
	}
	if e.ResponseType != "" {
		fmt.Fprintf(w, "response: %s\n", e.ResponseType)
	}
	if len(e.Orchestration) > 0 {
		fmt.Fprintf(w, "\norchestration (%d):\n", len(e.Orchestration))
		for _, s := range e.Orchestration {
			fmt.Fprintf(w, "  %s  %s\n", s.Name, loc(s.Location))
		}
	}
	if len(e.Upstreams) > 0 {
		fmt.Fprintf(w, "\nupstreams (%d):\n", len(e.Upstreams))
		for _, u := range e.Upstreams {
			fmt.Fprintf(w, "  %s %s  %s  [%s]\n", u.Method, u.URL, loc(u.Location), u.Confidence)
		}
	}
	return nil
}

func renderUpstreams(w io.Writer, u models.Upstreams) error {
	fmt.Fprintf(w, "upstreams in %s (%d)\n", u.Root, len(u.Calls))
	for _, c := range u.Calls {
		fmt.Fprintf(w, "  %s %s  decode=%s  %s  [%s]\n",
			c.Method, c.URL, c.DecodeType, loc(c.Location), c.Confidence)
	}
	return nil
}

func renderStruct(w io.Writer, s models.Struct, indent int) error {
	pad := strings.Repeat("  ", indent)
	fmt.Fprintf(w, "%s%s {\n", pad, s.Name)
	for _, f := range s.Fields {
		jsonName := f.JSONName
		if jsonName == "" {
			jsonName = f.Name
		}
		opt := ""
		if f.Optional {
			opt = "?"
		}
		fmt.Fprintf(w, "%s  %s%s: %s\n", pad, jsonName, opt, f.Type)
		if f.Nested != nil {
			_ = renderStruct(w, *f.Nested, indent+2)
		}
	}
	fmt.Fprintf(w, "%s}\n", pad)
	return nil
}
