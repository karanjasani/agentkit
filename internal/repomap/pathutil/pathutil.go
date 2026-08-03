// Package pathutil provides deterministic, cross-platform path and location
// helpers shared by the analysis packages.
package pathutil

import (
	"go/token"
	"path/filepath"
	"strings"

	"github.com/karanjasani/agentkit/pkg/models"
)

// Rel converts an absolute file path to a module-root-relative, forward-slash
// path so output is byte-identical across operating systems.
func Rel(moduleDir, file string) string {
	if file == "" {
		return ""
	}
	if moduleDir != "" {
		if rel, err := filepath.Rel(moduleDir, file); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(file)
}

// Loc builds a models.Location from a token position.
func Loc(fset *token.FileSet, moduleDir string, pos token.Pos) models.Location {
	if !pos.IsValid() {
		return models.Location{}
	}
	p := fset.Position(pos)
	return models.Location{
		File: Rel(moduleDir, p.Filename),
		Line: p.Line,
		Col:  p.Column,
	}
}
