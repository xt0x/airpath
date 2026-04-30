package awsintegration

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestStorageRepositoriesDoNotImportFlightAwareTransportDTOs(t *testing.T) {
	for _, filename := range []string{"dynamodb_positions.go", "s3_repository.go"} {
		t.Run(filename, func(t *testing.T) {
			parsed, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", filename, err)
			}
			for _, imported := range parsed.Imports {
				if strings.Contains(strings.Trim(imported.Path.Value, `"`), "/flightaware") {
					t.Fatalf("%s imports %s; FlightAware DTO translation must stay in the fetch adapter", filename, imported.Path.Value)
				}
			}
		})
	}
}
