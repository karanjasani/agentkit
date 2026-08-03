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

// Overview returns a high-level map of the module.
func Overview(ctx context.Context, l *loader.Loader) (models.Overview, error) {
	res, err := l.Load(ctx, loader.TierShallow, false)
	if err != nil {
		return models.Overview{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}
	if len(res.Packages) == 0 {
		return models.Overview{}, rerr.New(rerr.LoadFailed, true, "no Go packages found under %s", l.Dir)
	}

	out := models.Overview{
		Module:      res.ModulePath,
		GoVersion:   res.GoVersion,
		Packages:    []models.PkgRef{},
		Entrypoints: []models.PkgRef{},
		Generated:   []string{},
		Vendor:      []string{},
	}

	fileCount := 0
	seenVendor := map[string]bool{}
	seenGenerated := map[string]bool{}

	for _, p := range res.Packages {
		if p.PkgPath == "" {
			continue
		}
		dir := packageDir(res.ModuleDir, p)
		ref := models.PkgRef{ImportPath: p.PkgPath, Name: p.Name, Dir: dir}
		out.Packages = append(out.Packages, ref)

		files := p.GoFiles
		if len(files) == 0 {
			files = p.CompiledGoFiles
		}
		fileCount += len(files)

		for _, f := range files {
			rel := relPath(res.ModuleDir, f)
			if strings.Contains(rel, "/vendor/") || strings.HasPrefix(rel, "vendor/") {
				top := vendorDir(rel)
				if top != "" && !seenVendor[top] {
					seenVendor[top] = true
					out.Vendor = append(out.Vendor, top)
				}
			}
			if looksGenerated(rel) && !seenGenerated[rel] {
				seenGenerated[rel] = true
				out.Generated = append(out.Generated, rel)
			}
		}

		if p.Name == "main" {
			out.Entrypoints = append(out.Entrypoints, ref)
		}
	}

	sortPkgRefs(out.Packages)
	sortPkgRefs(out.Entrypoints)
	sort.Strings(out.Generated)
	sort.Strings(out.Vendor)

	out.Stats = models.Stats{
		Packages:    len(out.Packages),
		Files:       fileCount,
		Entrypoints: len(out.Entrypoints),
	}
	return out, nil
}

// packageDir returns the module-relative directory of a package.
func packageDir(moduleDir string, p *packages.Package) string {
	files := p.GoFiles
	if len(files) == 0 {
		files = p.CompiledGoFiles
	}
	if len(files) == 0 {
		return ""
	}
	d := relPath(moduleDir, files[0])
	if i := strings.LastIndex(d, "/"); i >= 0 {
		return d[:i]
	}
	return "."
}

func vendorDir(rel string) string {
	idx := strings.Index(rel, "vendor/")
	if idx < 0 {
		return ""
	}
	rest := rel[idx+len("vendor/"):]
	parts := strings.Split(rest, "/")
	if len(parts) == 0 {
		return "vendor"
	}
	// Return vendor/<first path segment> as a coarse grouping.
	return "vendor/" + parts[0]
}

// generatedSuffixes are common filename markers for machine-generated Go files.
// Filename-based detection keeps overview on the cheap shallow load tier.
var generatedSuffixes = []string{
	".pb.go",
	".pb.gw.go",
	"_generated.go",
	"_gen.go",
	"_string.go",
	".gen.go",
}

func looksGenerated(rel string) bool {
	base := rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		base = rel[i+1:]
	}
	if strings.HasPrefix(base, "zz_generated") || strings.HasPrefix(base, "bindata") {
		return true
	}
	for _, s := range generatedSuffixes {
		if strings.HasSuffix(base, s) {
			return true
		}
	}
	return false
}

func sortPkgRefs(refs []models.PkgRef) {
	sort.Slice(refs, func(i, j int) bool { return refs[i].ImportPath < refs[j].ImportPath })
}
