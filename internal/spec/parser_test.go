package spec

import (
	"os"
	"testing"
)

func TestParserWithPetstore(t *testing.T) {
	parser := NewParser()
	s, err := parser.Parse("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("expected successful parse, got error: %v", err)
	}

	if s.Info.Title != "Petstore Luxury API" {
		t.Errorf("expected title 'Petstore Luxury API', got '%s'", s.Info.Title)
	}

	if s.Info.Version != "1.2.0" {
		t.Errorf("expected version '1.2.0', got '%s'", s.Info.Version)
	}

	if len(s.Servers) != 3 {
		t.Errorf("expected 3 servers, got %d", len(s.Servers))
	}

	if len(s.Endpoints) < 5 {
		t.Errorf("expected at least 5 endpoints, got %d", len(s.Endpoints))
	}

	// Verify schemas in components
	if _, ok := s.Components.Schemas["Pet"]; !ok {
		t.Errorf("expected 'Pet' schema in components")
	}

	// Verify validator
	validator := NewValidator()
	diagnostics := validator.Validate(s)
	for _, d := range diagnostics {
		if d.Severity == "ERROR" {
			t.Errorf("unexpected validation error: %s - %s", d.Path, d.Message)
		}
	}
}

func TestParserInvalidContent(t *testing.T) {
	parser := NewParser()
	_, err := parser.ParseBytes([]byte("invalid: [yaml content: {"))
	if err == nil {
		t.Errorf("expected error for malformed yaml, got nil")
	}
}

func TestParserMissingFile(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse("non_existent_file.yaml")
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestValidatorMissingTitle(t *testing.T) {
	s := &Spec{
		Info: Info{Title: ""},
	}
	validator := NewValidator()
	diags := validator.Validate(s)
	found := false
	for _, d := range diags {
		if d.Severity == "ERROR" && d.Path == "info.title" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected validation error for missing title")
	}
}

func init() {
	// Ensure testdata exists when running in internal/spec
	if _, err := os.Stat("../../testdata/petstore.yaml"); err != nil {
		// If running from root:
		if _, errRoot := os.Stat("testdata/petstore.yaml"); errRoot == nil {
			_ = os.Chdir("../..")
		}
	}
}
