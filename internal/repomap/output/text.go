package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/karanjasani/agentkit/pkg/models"
)

// ew accumulates the first write error so the text renderers can emit many
// lines and check for failure once. After an error every further write is a
// no-op, which keeps the renderers readable and satisfies errcheck.
type ew struct {
	w   io.Writer
	err error
}

func (e *ew) printf(format string, a ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, a...)
}

func (e *ew) println(a ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintln(e.w, a...)
}

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
	e := &ew{w: w}
	e.printf("module: %s\n", o.Module)
	if o.GoVersion != "" {
		e.printf("go: %s\n", o.GoVersion)
	}
	e.printf("packages: %d  files: %d  entrypoints: %d\n",
		o.Stats.Packages, o.Stats.Files, o.Stats.Entrypoints)
	if len(o.Entrypoints) > 0 {
		e.println("\nentrypoints:")
		for _, p := range o.Entrypoints {
			e.printf("  %s (%s)\n", p.ImportPath, p.Dir)
		}
	}
	e.println("\npackages:")
	for _, p := range o.Packages {
		e.printf("  %s\n", p.ImportPath)
	}
	return e.err
}

func renderPackage(w io.Writer, p models.Package) error {
	e := &ew{w: w}
	e.printf("package %s (%s)\n", p.Name, p.ImportPath)
	e.printf("dir: %s\n", p.Dir)
	e.printf("\nimports (%d):\n", len(p.Imports))
	for _, i := range p.Imports {
		e.printf("  %s\n", i)
	}
	e.printf("\nimported by (%d):\n", len(p.ImportedBy))
	for _, i := range p.ImportedBy {
		e.printf("  %s\n", i)
	}
	e.printf("\nexports (%d):\n", len(p.Exports))
	for _, s := range p.Exports {
		e.printf("  %s %s  %s\n", s.Kind, s.Name, loc(s.Location))
	}
	if len(p.TestFiles) > 0 {
		e.printf("\ntest files (%d):\n", len(p.TestFiles))
		for _, f := range p.TestFiles {
			e.printf("  %s\n", f)
		}
	}
	return e.err
}

func renderSymbol(w io.Writer, s models.Symbol) error {
	e := &ew{w: w}
	e.printf("%s %s\n", s.Kind, s.Name)
	e.printf("package: %s\n", s.Package)
	e.printf("location: %s\n", loc(s.Location))
	if s.Signature != "" {
		e.printf("signature: %s\n", s.Signature)
	}
	if s.Doc != "" {
		e.printf("\ndoc:\n%s\n", s.Doc)
	}
	if s.Body != "" {
		e.printf("\nbody:\n%s\n", s.Body)
	}
	if s.Shape != nil {
		e.println("\nshape:")
		if e.err == nil {
			e.err = renderStruct(w, *s.Shape, 1)
		}
	}
	if len(s.Callers) > 0 {
		e.printf("\ncallers (%d):\n", len(s.Callers))
		for _, c := range s.Callers {
			e.printf("  %s  %s  [%s]\n", loc(c.Location), strings.TrimSpace(c.Context), c.Confidence)
		}
	}
	return e.err
}

func renderCallers(w io.Writer, c models.Callers) error {
	e := &ew{w: w}
	e.printf("callers of %s (%s)\n", c.Symbol, c.Package)
	e.printf("\ndirect (%d):\n", len(c.Direct))
	for _, d := range c.Direct {
		e.printf("  %s  %s  %s  [%s]\n", d.Symbol, loc(d.Location), strings.TrimSpace(d.Context), d.Confidence)
	}
	e.printf("\nindirect (%d):\n", len(c.Indirect))
	for _, d := range c.Indirect {
		e.printf("  %s  %s  [%s]\n", d.Symbol, loc(d.Location), d.Confidence)
	}
	return e.err
}

func renderDeps(w io.Writer, d models.Deps) error {
	e := &ew{w: w}
	e.printf("deps of %s (depth %d)\n", d.Root, d.Depth)
	e.printf("\nnodes (%d):\n", len(d.Nodes))
	for _, n := range d.Nodes {
		e.printf("  %s\n", n)
	}
	e.printf("\nedges (%d):\n", len(d.Edges))
	for _, edge := range d.Edges {
		e.printf("  %s -> %s\n", edge.From, edge.To)
	}
	return e.err
}

func renderImpact(w io.Writer, i models.Impact) error {
	e := &ew{w: w}
	e.printf("impact vs %s\n", i.Base)
	e.printf("risk: %d (%s)\n", i.RiskScore, i.RiskLevel)
	e.printf("\nchanged packages (%d):\n", len(i.ChangedPackages))
	for _, p := range i.ChangedPackages {
		e.printf("  %s\n", p)
	}
	e.printf("\naffected packages (%d):\n", len(i.AffectedPackages))
	for _, p := range i.AffectedPackages {
		e.printf("  %s\n", p)
	}
	e.printf("\nrecommended tests (%d):\n", len(i.RecommendedTests))
	for _, t := range i.RecommendedTests {
		e.printf("  %s\n", t)
	}
	return e.err
}

func renderTests(w io.Writer, t models.Tests) error {
	e := &ew{w: w}
	e.printf("tests for %s\n", t.Symbol)
	writeTestGroup(e, "unit", t.Unit)
	writeTestGroup(e, "integration", t.Integration)
	writeTestGroup(e, "benchmark", t.Benchmark)
	return e.err
}

func writeTestGroup(e *ew, label string, tests []models.Test) {
	e.printf("\n%s (%d):\n", label, len(tests))
	for _, t := range tests {
		e.printf("  %s  %s\n", t.Name, loc(t.Location))
	}
}

func renderEndpoint(w io.Writer, e models.Endpoint) error {
	out := &ew{w: w}
	out.printf("%s %s  [%s, %s]\n", e.Method, e.Path, e.Framework, e.Confidence)
	out.printf("route: %s\n", loc(e.Route))
	if e.Handler != nil {
		out.printf("handler: %s  %s\n", e.Handler.Name, loc(e.Handler.Location))
	}
	if e.RequestType != "" {
		out.printf("request: %s\n", e.RequestType)
	}
	if e.ResponseType != "" {
		out.printf("response: %s\n", e.ResponseType)
	}
	if len(e.Orchestration) > 0 {
		out.printf("\norchestration (%d):\n", len(e.Orchestration))
		for _, s := range e.Orchestration {
			out.printf("  %s  %s\n", s.Name, loc(s.Location))
		}
	}
	if len(e.Upstreams) > 0 {
		out.printf("\nupstreams (%d):\n", len(e.Upstreams))
		for _, u := range e.Upstreams {
			out.printf("  %s %s  %s  [%s]\n", u.Method, u.URL, loc(u.Location), u.Confidence)
		}
	}
	return out.err
}

func renderUpstreams(w io.Writer, u models.Upstreams) error {
	e := &ew{w: w}
	e.printf("upstreams in %s (%d)\n", u.Root, len(u.Calls))
	for _, c := range u.Calls {
		e.printf("  %s %s  decode=%s  %s  [%s]\n",
			c.Method, c.URL, c.DecodeType, loc(c.Location), c.Confidence)
	}
	return e.err
}

func renderStruct(w io.Writer, s models.Struct, indent int) error {
	e := &ew{w: w}
	pad := strings.Repeat("  ", indent)
	e.printf("%s%s {\n", pad, s.Name)
	for _, f := range s.Fields {
		jsonName := f.JSONName
		if jsonName == "" {
			jsonName = f.Name
		}
		opt := ""
		if f.Optional {
			opt = "?"
		}
		e.printf("%s  %s%s: %s\n", pad, jsonName, opt, f.Type)
		if f.Nested != nil && e.err == nil {
			e.err = renderStruct(w, *f.Nested, indent+2)
		}
	}
	e.printf("%s}\n", pad)
	return e.err
}
