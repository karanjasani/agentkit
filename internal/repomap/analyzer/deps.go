package analyzer

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// DepsGraph returns the intra-module dependency graph reachable from a root
// package, along with the maximum dependency depth.
func DepsGraph(ctx context.Context, l *loader.Loader, path string) (models.Deps, error) {
	res, err := l.Load(ctx, loader.TierShallow, false)
	if err != nil {
		return models.Deps{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	root := resolvePackage(res, path)
	if root == nil {
		return models.Deps{}, rerr.New(rerr.PackageNotFound, true, "package not found: %s", path)
	}

	// Index all module packages by import path.
	index := map[string]*packages.Package{}
	for _, p := range res.Packages {
		if p.PkgPath == "" || isTestVariant(p.PkgPath) {
			continue
		}
		if strings.HasPrefix(p.PkgPath, res.ModulePath) {
			index[p.PkgPath] = p
		}
	}

	nodes := map[string]bool{}
	edges := map[string]bool{}
	edgeList := []models.DepEdge{}
	depth := map[string]int{root.PkgPath: 0}

	queue := []string{root.PkgPath}
	nodes[root.PkgPath] = true
	maxDepth := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		p := index[cur]
		if p == nil {
			continue
		}
		// Deterministic neighbor order.
		imports := make([]string, 0, len(p.Imports))
		for ip := range p.Imports {
			imports = append(imports, ip)
		}
		sort.Strings(imports)

		for _, ip := range imports {
			if _, ok := index[ip]; !ok {
				continue // external dependency: not expanded
			}
			ek := cur + "\x00" + ip
			if !edges[ek] {
				edges[ek] = true
				edgeList = append(edgeList, models.DepEdge{From: cur, To: ip})
			}
			if !nodes[ip] {
				nodes[ip] = true
				depth[ip] = depth[cur] + 1
				if depth[ip] > maxDepth {
					maxDepth = depth[ip]
				}
				queue = append(queue, ip)
			}
		}
	}

	nodeList := make([]string, 0, len(nodes))
	for n := range nodes {
		nodeList = append(nodeList, n)
	}
	sort.Strings(nodeList)
	sort.Slice(edgeList, func(i, j int) bool {
		if edgeList[i].From != edgeList[j].From {
			return edgeList[i].From < edgeList[j].From
		}
		return edgeList[i].To < edgeList[j].To
	})

	return models.Deps{
		Root:  root.PkgPath,
		Depth: maxDepth,
		Nodes: nodeList,
		Edges: edgeList,
	}, nil
}
