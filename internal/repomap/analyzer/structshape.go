package analyzer

import (
	"context"
	"go/types"
	"reflect"
	"strings"

	"github.com/karanjasani/agentkit/internal/repomap/loader"
	"github.com/karanjasani/agentkit/internal/repomap/rerr"
	"github.com/karanjasani/agentkit/pkg/models"
)

// maxShapeDepth caps recursion into nested struct types to keep output bounded.
const maxShapeDepth = 6

// StructShape returns the recursive JSON contract of a named struct type.
func StructShape(ctx context.Context, l *loader.Loader, name string) (models.Struct, error) {
	res, err := l.Load(ctx, loader.TierTypes, false)
	if err != nil {
		return models.Struct{}, rerr.New(rerr.LoadFailed, false, "%v", err)
	}

	obj := findTypeName(res, name)
	if obj == nil {
		return models.Struct{}, rerr.New(rerr.TypeNotFound, true, "type not found: %s", name)
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return models.Struct{}, rerr.New(rerr.NotAStruct, true, "%s is not a named type", name)
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return models.Struct{}, rerr.New(rerr.NotAStruct, true, "%s is not a struct type", name)
	}
	return buildStruct(obj.Name(), st, 0, map[string]bool{}), nil
}

// findTypeName locates a *types.TypeName by bare name or "pkg.Name".
func findTypeName(res *loader.Result, name string) *types.TypeName {
	pkgQual, symName := splitQualified(name)
	for _, p := range res.Packages {
		if p.Types == nil || isTestVariant(p.PkgPath) {
			continue
		}
		if pkgQual != "" && p.Types.Name() != pkgQual {
			continue
		}
		obj := p.Types.Scope().Lookup(symName)
		if obj == nil {
			continue
		}
		if tn, ok := obj.(*types.TypeName); ok {
			return tn
		}
	}
	return nil
}

// splitQualified splits "pkg.Name" into ("pkg", "Name"); a bare name yields
// ("", name). "Type.Method" is handled by callers separately.
func splitQualified(name string) (pkg, sym string) {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "", name
}

func buildStruct(name string, st *types.Struct, depth int, seen map[string]bool) models.Struct {
	out := models.Struct{Name: name, Fields: []models.Field{}}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))
		jsonName, omit, skip := parseJSONTag(tag, f.Name())
		if skip {
			continue
		}
		if !f.Exported() && !f.Embedded() {
			continue
		}
		field := models.Field{
			Name:     f.Name(),
			JSONName: jsonName,
			Type:     typeString(f.Type()),
			Optional: omit,
		}
		if nested := nestedStruct(f.Type()); nested != nil && depth < maxShapeDepth {
			key := typeString(f.Type())
			if !seen[key] {
				seen[key] = true
				ns := buildStruct(structTypeName(f.Type()), nested, depth+1, seen)
				field.Nested = &ns
				delete(seen, key)
			}
		}
		out.Fields = append(out.Fields, field)
	}
	return out
}

// parseJSONTag returns the effective JSON name, whether the field is omitempty,
// and whether it is skipped ("-").
func parseJSONTag(tag reflect.StructTag, fieldName string) (name string, omit, skip bool) {
	v := tag.Get("json")
	if v == "" {
		return fieldName, false, false
	}
	parts := strings.Split(v, ",")
	if parts[0] == "-" && len(parts) == 1 {
		return "", false, true
	}
	name = parts[0]
	if name == "" {
		name = fieldName
	}
	for _, o := range parts[1:] {
		if o == "omitempty" {
			omit = true
		}
	}
	return name, omit, false
}

// nestedStruct unwraps pointers/slices/arrays/maps to reach a struct type, if any.
func nestedStruct(t types.Type) *types.Struct {
	switch u := t.(type) {
	case *types.Pointer:
		return nestedStruct(u.Elem())
	case *types.Slice:
		return nestedStruct(u.Elem())
	case *types.Array:
		return nestedStruct(u.Elem())
	case *types.Named:
		if s, ok := u.Underlying().(*types.Struct); ok {
			return s
		}
	case *types.Struct:
		return u
	}
	return nil
}

func structTypeName(t types.Type) string {
	switch u := t.(type) {
	case *types.Pointer:
		return structTypeName(u.Elem())
	case *types.Slice:
		return structTypeName(u.Elem())
	case *types.Array:
		return structTypeName(u.Elem())
	case *types.Named:
		return u.Obj().Name()
	default:
		return typeString(t)
	}
}

// typeString renders a type using unqualified package names for stable output.
func typeString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
