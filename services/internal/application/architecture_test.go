package application

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplicationPackageDoesNotImportFlightAwareAdapter(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob application files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imported := range parsed.Imports {
			if strings.Contains(strings.Trim(imported.Path.Value, `"`), "/flightaware") {
				t.Fatalf("%s imports %s; application ports must use application-owned DTOs", file, imported.Path.Value)
			}
		}
	}
}

func TestFlightDetailResponseUsesPublicContractDTO(t *testing.T) {
	responseType := lookupStructType(t, "types.go", "FlightDetailResponse")
	flightField := lookupField(t, responseType, "Flight")
	if got := exprString(flightField.Type); got != "FlightDetail" {
		t.Fatalf("FlightDetailResponse.Flight type = %s, want FlightDetail", got)
	}
}

func lookupStructType(t *testing.T, filename string, typeName string) *ast.StructType {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	for _, decl := range parsed.Decls {
		generic, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range generic.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != typeName {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s is %T, want struct", typeName, typeSpec.Type)
			}
			return structType
		}
	}
	t.Fatalf("type %s not found in %s", typeName, filename)
	return nil
}

func lookupField(t *testing.T, structType *ast.StructType, fieldName string) *ast.Field {
	t.Helper()
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name == fieldName {
				return field
			}
		}
	}
	t.Fatalf("field %s not found", fieldName)
	return nil
}

func exprString(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.SelectorExpr:
		return exprString(typed.X) + "." + typed.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(typed.X)
	default:
		return "<unsupported>"
	}
}
