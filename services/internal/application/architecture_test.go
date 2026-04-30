package application

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
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

func TestPublicAPIDTOJSONFieldsMatchSharedTypesContract(t *testing.T) {
	source, err := os.ReadFile("../../../packages/shared-types/src/api-types.ts")
	if err != nil {
		t.Fatalf("read shared API types: %v", err)
	}
	sharedTypes := string(source)
	cases := []struct {
		goStruct string
		tsType   string
	}{
		{goStruct: "CacheMetadata", tsType: "CacheMetadata"},
		{goStruct: "FlightSummaryItem", tsType: "FlightSummaryItem"},
		{goStruct: "FlightDetail", tsType: "FlightDetail"},
		{goStruct: "MapLayer", tsType: "MapLayer"},
		{goStruct: "Position", tsType: "Position"},
		{goStruct: "FlightSearchResponse", tsType: "FlightSearchResponse"},
		{goStruct: "FlightDetailResponse", tsType: "FlightDetailResponse"},
		{goStruct: "FlightPositionsResponse", tsType: "FlightPositionsResponse"},
		{goStruct: "FlightMapDataResponse", tsType: "FlightMapDataResponse"},
		{goStruct: "FetchTask", tsType: "FetchTask"},
		{goStruct: "FlightRefreshResponse", tsType: "FlightRefreshResponse"},
		{goStruct: "UsageStatus", tsType: "UsageStatusResponse"},
		{goStruct: "APIError", tsType: "ApiError"},
	}
	for _, testCase := range cases {
		t.Run(testCase.goStruct, func(t *testing.T) {
			goFields := goContractFields(t, "types.go", testCase.goStruct)
			tsFields := tsContractFields(t, sharedTypes, testCase.tsType)
			if diff := contractFieldDiff(goFields, tsFields); diff != "" {
				t.Fatalf("%s JSON contract differs from shared %s: %s", testCase.goStruct, testCase.tsType, diff)
			}
		})
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

type contractField struct {
	Type     string
	Required bool
	Nullable bool
}

func goContractFields(t *testing.T, filename string, typeName string) map[string]contractField {
	t.Helper()
	structType := lookupStructType(t, filename, typeName)
	fields := map[string]contractField{}
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := strings.Trim(field.Tag.Value, "`")
		jsonTag := reflect.StructTag(tag).Get("json")
		tagParts := strings.Split(jsonTag, ",")
		name := tagParts[0]
		if name == "" || name == "-" {
			continue
		}
		fields[name] = contractField{
			Type:     goJSONType(field.Type),
			Required: !contains(tagParts[1:], "omitempty"),
			Nullable: goJSONNullable(field.Type),
		}
	}
	return fields
}

func tsContractFields(t *testing.T, source string, typeName string) map[string]contractField {
	t.Helper()
	if block, ok := extractTSInterfaceBlock(source, typeName); ok {
		return tsFieldsFromBlock(block)
	}
	if block, ok := extractTSPickBlock(source, typeName); ok {
		return tsFieldsFromPick(block)
	}
	t.Fatalf("shared TypeScript type %s not found", typeName)
	return nil
}

func extractTSInterfaceBlock(source string, typeName string) (string, bool) {
	prefix := "export interface " + typeName
	start := indexTSDeclaration(source, prefix)
	if start == -1 {
		return "", false
	}
	open := strings.Index(source[start:], "{")
	if open == -1 {
		return "", false
	}
	open += start
	close := matchingSourceBrace(source, open)
	if close == -1 {
		return "", false
	}
	return source[open+1 : close], true
}

func extractTSPickBlock(source string, typeName string) (string, bool) {
	prefix := "export type " + typeName + " = Pick<"
	start := indexTSDeclaration(source, prefix)
	if start == -1 {
		return "", false
	}
	end := strings.Index(source[start:], ">;")
	if end == -1 {
		return "", false
	}
	return source[start : start+end], true
}

func indexTSDeclaration(source string, prefix string) int {
	start := 0
	for {
		index := strings.Index(source[start:], prefix)
		if index == -1 {
			return -1
		}
		index += start
		after := source[index+len(prefix)]
		if after == ' ' || after == '\n' || after == '{' || after == '<' {
			return index
		}
		start = index + len(prefix)
	}
}

func tsFieldsFromBlock(block string) map[string]contractField {
	pattern := regexp.MustCompile(`(?m)^\s*([A-Za-z][A-Za-z0-9]*)(\?)?:\s*([^;\n]+)`)
	fields := map[string]contractField{}
	depth := 0
	for _, line := range strings.Split(block, "\n") {
		if depth == 0 {
			if match := pattern.FindStringSubmatch(line); match != nil {
				fields[match[1]] = tsContractField(match[2] != "", match[3])
			}
		}
		depth += strings.Count(line, "{")
		depth -= strings.Count(line, "}")
		if depth < 0 {
			depth = 0
		}
	}
	return fields
}

func tsFieldsFromPick(block string) map[string]contractField {
	pattern := regexp.MustCompile(`"([A-Za-z][A-Za-z0-9]*)"`)
	matches := pattern.FindAllStringSubmatch(block, -1)
	fields := make(map[string]contractField, len(matches))
	for _, match := range matches {
		fields[match[1]] = contractField{Required: true}
	}
	return fields
}

func matchingSourceBrace(source string, open int) int {
	depth := 0
	for index := open; index < len(source); index++ {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func contractFieldDiff(goFields map[string]contractField, tsFields map[string]contractField) string {
	goNames := sortedFieldNames(goFields)
	tsNames := sortedFieldNames(tsFields)
	if strings.Join(goNames, ",") != strings.Join(tsNames, ",") {
		return "field names: Go " + strings.Join(goNames, ",") + "; TypeScript " + strings.Join(tsNames, ",")
	}
	for _, name := range goNames {
		goField := goFields[name]
		tsField := tsFields[name]
		if tsField.Type == "" {
			continue
		}
		if goField != tsField {
			return name + ": Go " + describeContractField(goField) + "; TypeScript " + describeContractField(tsField)
		}
	}
	return ""
}

func sortedFieldNames(fields map[string]contractField) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func describeContractField(field contractField) string {
	required := "optional"
	if field.Required {
		required = "required"
	}
	nullable := "non-null"
	if field.Nullable {
		nullable = "nullable"
	}
	return field.Type + "/" + required + "/" + nullable
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func goJSONType(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return goIdentJSONType(typed.Name)
	case *ast.SelectorExpr:
		return goSelectorJSONType(exprString(typed))
	case *ast.StarExpr:
		return goJSONType(typed.X)
	case *ast.ArrayType:
		return "array"
	case *ast.MapType:
		return "object"
	default:
		return "object"
	}
}

func goIdentJSONType(name string) string {
	switch name {
	case "string":
		return "string"
	case "int", "int64", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "CacheFreshness", "CacheSource", "FetchTaskType", "FetchReason", "MapSource", "ApiErrorCode":
		return "string"
	default:
		return "object"
	}
}

func goSelectorJSONType(name string) string {
	switch name {
	case "domain.FlightID", "domain.FlightIDType", "domain.ProvisionalFlightLegID", "domain.FAFlightID", "domain.AirportCode", "domain.ISODateTimeString", "domain.PositionSource", "FetchTaskType", "FetchReason", "ApiErrorCode", "CacheFreshness", "CacheSource", "MapSource":
		return "string"
	default:
		return "object"
	}
}

func goJSONNullable(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.StarExpr, *ast.MapType:
		return true
	default:
		return false
	}
}

func tsContractField(optional bool, rawType string) contractField {
	return contractField{
		Type:     tsJSONType(rawType),
		Required: !optional,
		Nullable: tsJSONNullable(rawType),
	}
}

func tsJSONType(rawType string) string {
	normalized := strings.TrimSpace(rawType)
	normalized = strings.TrimSuffix(normalized, ";")
	normalized = strings.ReplaceAll(normalized, " | null", "")
	normalized = strings.ReplaceAll(normalized, "null | ", "")
	normalized = strings.TrimSpace(normalized)
	switch {
	case normalized == "string", strings.HasPrefix(normalized, "\""):
		return "string"
	case normalized == "number":
		return "number"
	case normalized == "boolean":
		return "boolean"
	case normalized == "1":
		return "number"
	case strings.HasSuffix(normalized, "[]"), strings.HasPrefix(normalized, "Array<"):
		return "array"
	case normalized == "FlightID", normalized == "FlightIDType", normalized == "ProvisionalFlightLegID", normalized == "FAFlightID", normalized == "ISODateTimeString", normalized == "PositionSource", normalized == "FetchTaskType", normalized == "ApiErrorCode", normalized == "CacheFreshness", normalized == "CacheSource", normalized == "MapSource":
		return "string"
	default:
		return "object"
	}
}

func tsJSONNullable(rawType string) bool {
	for _, part := range strings.Split(rawType, "|") {
		if strings.TrimSpace(part) == "null" {
			return true
		}
	}
	return false
}
