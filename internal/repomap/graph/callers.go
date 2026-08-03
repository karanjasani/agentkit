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
	"github.com/karanjasani/agentkit/internal/repomap/srcline"
	"github.com/karanjasani/agentkit/pkg/models"
)

// Callers returns the direct and indirect callers of a symbol.
func Callers(ctx context.Context, l *loader.Loader, name string) (models.Callers, error) {
	res, err := l.Load(ctx, loader.TierTypes, false)
	if err != nil {
		return models.Callers{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	targets := funcObjects(res, name)
	if len(targets) == 0 {
		return models.Callers{}, rerr.New(rerr.SymbolNotFound, true, "symbol not found: %s", name)
	}

	prog, _ := ssautil.AllPackages(res.Packages, ssa.InstantiateGenerics)
	prog.Build()

	// Map target type objects to SSA functions.
	targetFns := map[*ssa.Function]bool{}
	for _, obj := range targets {
		if fn := prog.FuncValue(obj); fn != nil {
			targetFns[fn] = true
		}
	}
	if len(targetFns) == 0 {
		// The symbol exists but has no SSA function (e.g. only a declaration).
		primary := targets[0]
		return models.Callers{
			Symbol:   primary.Name(),
			Package:  pkgPath(primary),
			Location: pathutil.Loc(res.Fset, res.ModuleDir, primary.Pos()),
			Direct:   []models.Caller{},
			Indirect: []models.Caller{},
		}, nil
	}

	cg := cha.CallGraph(prog)
	cg.DeleteSyntheticNodes()

	lines := srcline.New()

	directSet := map[string]models.Caller{}
	directNodes := map[*callgraph.Node]bool{}

	for fn := range targetFns {
		node := cg.Nodes[fn]
		if node == nil {
			continue
		}
		for _, edge := range node.In {
			caller := edge.Caller.Func
			if caller == nil || targetFns[caller] {
				continue
			}
			if !inModule(res, caller) {
				continue
			}
			c := callerFromEdge(res, lines, edge)
			directSet[callerKey(c)] = c
			directNodes[edge.Caller] = true
		}
	}

	// Indirect callers: callers of the direct callers (one extra hop), excluding
	// direct ones and the target itself.
	indirectSet := map[string]models.Caller{}
	for node := range directNodes {
		for _, edge := range node.In {
			caller := edge.Caller.Func
			if caller == nil || targetFns[caller] {
				continue
			}
			if !inModule(res, caller) {
				continue
			}
			c := callerFromEdge(res, lines, edge)
			key := callerKey(c)
			if _, isDirect := directSet[key]; isDirect {
				continue
			}
			indirectSet[key] = c
		}
	}

	primary := targets[0]
	out := models.Callers{
		Symbol:   primary.Name(),
		Package:  pkgPath(primary),
		Location: pathutil.Loc(res.Fset, res.ModuleDir, primary.Pos()),
		Direct:   sortedCallers(directSet),
		Indirect: sortedCallers(indirectSet),
	}
	return out, nil
}

func callerFromEdge(res *loader.Result, lines *srcline.Cache, edge *callgraph.Edge) models.Caller {
	caller := edge.Caller.Func
	var loc models.Location
	context := ""
	confidence := "possible"

	if edge.Site != nil {
		loc = pathutil.Loc(res.Fset, res.ModuleDir, edge.Site.Pos())
		if edge.Site.Common().StaticCallee() != nil {
			confidence = "direct"
		}
		pos := res.Fset.Position(edge.Site.Pos())
		context = lines.Line(pos.Filename, pos.Line)
	} else {
		loc = pathutil.Loc(res.Fset, res.ModuleDir, caller.Pos())
	}

	return models.Caller{
		Symbol:     caller.Name(),
		Package:    fnPkgPath(caller),
		Location:   loc,
		Context:    context,
		Confidence: confidence,
	}
}

func callerKey(c models.Caller) string {
	return c.Package + "|" + c.Symbol + "|" + c.Location.File + "|" +
		itoa(c.Location.Line)
}

func sortedCallers(m map[string]models.Caller) []models.Caller {
	out := make([]models.Caller, 0, len(m))
	for _, c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Location.File != out[j].Location.File {
			return out[i].Location.File < out[j].Location.File
		}
		if out[i].Location.Line != out[j].Location.Line {
			return out[i].Location.Line < out[j].Location.Line
		}
		return out[i].Symbol < out[j].Symbol
	})
	return out
}

func fnPkgPath(fn *ssa.Function) string {
	if fn.Pkg != nil && fn.Pkg.Pkg != nil {
		return fn.Pkg.Pkg.Path()
	}
	return ""
}

// inModule reports whether an SSA function belongs to the module under
// analysis. Callers restricts its results to the main module so that
// over-approximated CHA edges into third-party dependencies (which also leak
// absolute module-cache paths) are never reported.
func inModule(res *loader.Result, fn *ssa.Function) bool {
	if res.ModulePath == "" {
		// Without a module path we cannot distinguish first- from third-party
		// code; fall back to reporting everything rather than nothing.
		return true
	}
	path := fnPkgPath(fn)
	if path == "" {
		return false
	}
	return path == res.ModulePath || strings.HasPrefix(path, res.ModulePath+"/")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
