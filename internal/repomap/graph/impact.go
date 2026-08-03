package graph

import (
	"context"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/git"
	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/pathutil"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// ChangeImpact computes the blast radius of the working tree versus a base ref.
func ChangeImpact(ctx context.Context, l *loader.Loader, dir, base string) (models.Impact, error) {
	repoRoot, err := git.RepoRoot(ctx, dir)
	if err != nil {
		return models.Impact{}, err
	}
	changed, err := git.ChangedFiles(ctx, dir, base)
	if err != nil {
		return models.Impact{}, err
	}

	res, err := l.Load(ctx, loader.TierTypes, true)
	if err != nil {
		return models.Impact{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	// Changed files come from git as repo-root-relative paths; the module may sit
	// in a subdirectory of the repo. Convert both to module-relative paths so
	// comparisons are robust to symlinks (e.g. macOS /var vs /private/var).
	changedRelSet := changedModuleRelative(repoRoot, res.ModuleDir, changed)
	changedRel := keys(changedRelSet)

	// Map changed files to their packages by module-relative path.
	changedPkgs := map[string]bool{}
	for _, p := range modulePackages(res) {
		if isTestVariant(p.PkgPath) {
			continue
		}
		for _, f := range allGoFiles(p) {
			if changedRelSet[pathutil.Rel(res.ModuleDir, f)] {
				changedPkgs[p.PkgPath] = true
				break
			}
		}
	}

	// Reverse dependency closure -> affected packages.
	reverse := buildReverseImports(res)
	affected := map[string]bool{}
	var queue []string
	for pkg := range changedPkgs {
		queue = append(queue, pkg)
	}
	sort.Strings(queue)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, importer := range reverse[cur] {
			if !affected[importer] && !changedPkgs[importer] {
				affected[importer] = true
				queue = append(queue, importer)
			}
		}
	}

	// Public API changed: exported symbols defined in changed files.
	apiChanged := publicAPIChanged(res, changedRelSet, changedPkgs)

	// Recommended tests: test files in changed + affected packages.
	recTests := recommendedTests(res, changedPkgs, affected)

	changedList := keys(changedPkgs)
	affectedList := keys(affected)

	score, level := riskScore(len(changedList), len(affectedList), len(apiChanged))

	return models.Impact{
		Base:             baseOr(base),
		ChangedFiles:     changedRel,
		ChangedPackages:  changedList,
		AffectedPackages: affectedList,
		PublicAPIChanged: apiChanged,
		RecommendedTests: recTests,
		RiskScore:        score,
		RiskLevel:        level,
	}, nil
}

// changedModuleRelative converts repo-root-relative changed paths into a set of
// module-root-relative paths, dropping files outside the module. Symlinks are
// resolved on both roots so the comparison is stable across platforms.
func changedModuleRelative(repoRoot, moduleDir string, changed []string) map[string]bool {
	repoEval := evalSymlinks(repoRoot)
	modEval := evalSymlinks(moduleDir)

	modRel := ""
	if rel, err := filepath.Rel(repoEval, modEval); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		modRel = filepath.ToSlash(rel)
	}

	out := map[string]bool{}
	for _, c := range changed {
		c = filepath.ToSlash(c)
		if modRel == "" {
			out[c] = true
			continue
		}
		if c == modRel {
			continue
		}
		if strings.HasPrefix(c, modRel+"/") {
			out[strings.TrimPrefix(c, modRel+"/")] = true
		}
	}
	return out
}

func evalSymlinks(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

func allGoFiles(p *packages.Package) []string {
	if len(p.GoFiles) > 0 {
		return p.GoFiles
	}
	return p.CompiledGoFiles
}

func buildReverseImports(res *loader.Result) map[string][]string {
	rev := map[string][]string{}
	seen := map[string]bool{}
	for _, p := range modulePackages(res) {
		if isTestVariant(p.PkgPath) {
			continue
		}
		for imp := range p.Imports {
			key := imp + "\x00" + p.PkgPath
			if seen[key] {
				continue
			}
			seen[key] = true
			rev[imp] = append(rev[imp], p.PkgPath)
		}
	}
	for k := range rev {
		sort.Strings(rev[k])
	}
	return rev
}

func publicAPIChanged(res *loader.Result, changedRelSet map[string]bool, changedPkgs map[string]bool) []models.Symbol {
	var out []models.Symbol
	for _, p := range modulePackages(res) {
		if !changedPkgs[p.PkgPath] || p.Types == nil {
			continue
		}
		scope := p.Types.Scope()
		for _, n := range scope.Names() {
			obj := scope.Lookup(n)
			if obj == nil || !obj.Exported() {
				continue
			}
			pos := res.Fset.Position(obj.Pos())
			if !changedRelSet[pathutil.Rel(res.ModuleDir, pos.Filename)] {
				continue
			}
			out = append(out, models.Symbol{
				Name:     obj.Name(),
				Kind:     objKind(obj),
				Package:  p.PkgPath,
				Location: pathutil.Loc(res.Fset, res.ModuleDir, obj.Pos()),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func recommendedTests(res *loader.Result, changedPkgs, affected map[string]bool) []string {
	set := map[string]bool{}
	for _, p := range modulePackages(res) {
		base := strings.TrimSuffix(p.PkgPath, ".test")
		base = strings.Split(base, " ")[0]
		if !changedPkgs[base] && !affected[base] {
			continue
		}
		for _, f := range p.GoFiles {
			rel := pathutil.Rel(res.ModuleDir, f)
			if strings.HasSuffix(rel, "_test.go") {
				set[rel] = true
			}
		}
	}
	out := keys(set)
	return out
}

// riskScore is a documented, deterministic function of the change surface.
//
//	score = min(100, 10*changedPkgs + 5*affectedPkgs + 8*publicAPIChanged)
//	level = low (<30), medium (<70), high (>=70)
func riskScore(changedPkgs, affectedPkgs, apiChanged int) (int, string) {
	score := 10*changedPkgs + 5*affectedPkgs + 8*apiChanged
	if score > 100 {
		score = 100
	}
	switch {
	case score < 30:
		return score, "low"
	case score < 70:
		return score, "medium"
	default:
		return score, "high"
	}
}

func objKind(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Func:
		if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
			return "method"
		}
		return "func"
	case *types.TypeName:
		return "type"
	case *types.Var:
		return "var"
	case *types.Const:
		return "const"
	default:
		return "symbol"
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func baseOr(base string) string {
	if base == "" {
		return "HEAD"
	}
	return base
}
