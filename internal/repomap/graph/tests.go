package graph

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/pathutil"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// RelatedTests returns tests that transitively exercise a symbol, grouped by
// kind (unit, integration, benchmark).
func RelatedTests(ctx context.Context, l *loader.Loader, name string) (models.Tests, error) {
	res, err := l.Load(ctx, loader.TierTypes, true)
	if err != nil {
		return models.Tests{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	targets := funcObjects(res, name)
	if len(targets) == 0 {
		return models.Tests{}, rerr.New(rerr.SymbolNotFound, true, "symbol not found: %s", name)
	}

	prog, _ := ssautil.AllPackages(res.Packages, ssa.InstantiateGenerics)
	prog.Build()

	targetFns := map[*ssa.Function]bool{}
	for _, obj := range targets {
		if fn := prog.FuncValue(obj); fn != nil {
			targetFns[fn] = true
		}
	}

	out := models.Tests{
		Symbol:      name,
		Unit:        []models.Test{},
		Integration: []models.Test{},
		Benchmark:   []models.Test{},
	}
	if len(targetFns) == 0 {
		return out, nil
	}

	cg := cha.CallGraph(prog)

	for fn, node := range cg.Nodes {
		if fn == nil || node == nil {
			continue
		}
		kind := testKind(res, fn)
		if kind == "" {
			continue
		}
		if !reaches(node, targetFns) {
			continue
		}
		t := models.Test{
			Name:     fn.Name(),
			Package:  fnPkgPath(fn),
			Location: pathutil.Loc(res.Fset, res.ModuleDir, fn.Pos()),
			Kind:     kind,
		}
		switch kind {
		case "benchmark":
			out.Benchmark = append(out.Benchmark, t)
		case "integration":
			out.Integration = append(out.Integration, t)
		default:
			out.Unit = append(out.Unit, t)
		}
	}

	sortTests(out.Unit)
	sortTests(out.Integration)
	sortTests(out.Benchmark)
	return out, nil
}

// testKind classifies a function as a test, returning "unit", "integration",
// "benchmark", or "" if it is not a test function.
func testKind(res *loader.Result, fn *ssa.Function) string {
	name := fn.Name()
	pos := res.Fset.Position(fn.Pos())
	if !strings.HasSuffix(pos.Filename, "_test.go") {
		return ""
	}
	switch {
	case strings.HasPrefix(name, "Benchmark"):
		return "benchmark"
	case strings.HasPrefix(name, "Test"):
		file := strings.ToLower(pos.Filename)
		if strings.Contains(file, "integration") || strings.Contains(strings.ToLower(name), "integration") {
			return "integration"
		}
		return "unit"
	default:
		return ""
	}
}

// reaches reports whether any target is reachable from start via call edges.
func reaches(start *callgraph.Node, targets map[*ssa.Function]bool) bool {
	visited := map[*callgraph.Node]bool{start: true}
	queue := []*callgraph.Node{start}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, e := range n.Out {
			callee := e.Callee
			if callee == nil || visited[callee] {
				continue
			}
			if callee.Func != nil && targets[callee.Func] {
				return true
			}
			visited[callee] = true
			queue = append(queue, callee)
		}
	}
	return false
}

func sortTests(ts []models.Test) {
	sort.Slice(ts, func(i, j int) bool {
		if ts[i].Location.File != ts[j].Location.File {
			return ts[i].Location.File < ts[j].Location.File
		}
		if ts[i].Name != ts[j].Name {
			return ts[i].Name < ts[j].Name
		}
		return ts[i].Package < ts[j].Package
	})
}
