// Package analyzer implements the syntax- and type-level repomap commands:
// overview, package, symbol, struct and deps. It operates over packages loaded
// by the loader package and returns models types.
package analyzer

import (
	"go/token"

	"github.com/karanjasani/agentkit/internal/repomap/pathutil"
	"github.com/karanjasani/agentkit/pkg/models"
)

// relPath converts an absolute file path to a module-root-relative path.
func relPath(moduleDir, file string) string {
	return pathutil.Rel(moduleDir, file)
}

// location builds a Location from a token position.
func location(fset *token.FileSet, moduleDir string, pos token.Pos) models.Location {
	return pathutil.Loc(fset, moduleDir, pos)
}
