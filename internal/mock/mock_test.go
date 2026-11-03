package mock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

func loadTestSpec(t *testing.T) *spec.Spec {
	t.Helper()
	p := spec.NewParser()
	s, err := p.Parse("../../testdata/petstore.yaml")
	if err != nil {
		// Fallback path if run from root
		s, err = p.Parse("testdata/petstore.yaml")
		if err != nil {
			t.Fatalf("failed to load test spec: %v", err)
		}
	}
	return s
}

func TestMockRouteMatchingAndExample(t *testing.T) {
	s := loadTestSpec(t)
	server := NewServer(Config{Port: 0, Spec: s})

	// Test GET /pets/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d with preferred example
	req := httptest.NewRequest(http.MethodGet, "/pets/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d?mock_example=goldenRetriever", nil)
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var pet map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pet); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if pet["name"] != "Buddy" {
		t.Errorf("expected pet name 'Buddy', got %v", pet["name"])
	}

	if pet["tag"] != "dog" {
		t.Errorf("expected tag 'dog', got %v", pet["tag"])
	}
}

func TestMockStatusOverride404(t *testing.T) {
	s := loadTestSpec(t)
	server := NewServer(Config{Port: 0, Spec: s})

	// Test Prefer: code=404
	req := httptest.NewRequest(http.MethodGet, "/pets/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d", nil)
	req.Header.Set("Prefer", "code=404")
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var errResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if errResp["code"] == nil {
		t.Errorf("expected 'code' field in error response")
	}
}

func TestMockCORSPreflight(t *testing.T) {
	s := loadTestSpec(t)
	server := NewServer(Config{Port: 0, Spec: s})

	req := httptest.NewRequest(http.MethodOptions, "/pets", nil)
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for OPTIONS, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *")
	}
}

func TestMockSyntheticGeneration(t *testing.T) {
	s := loadTestSpec(t)
	server := NewServer(Config{Port: 0, Spec: s})

	// GET /pets (returns array of Pet)
	req := httptest.NewRequest(http.MethodGet, "/pets", nil)
	rec := httptest.NewRecorder()

	server.handleRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var pets []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &pets); err != nil {
		t.Fatalf("failed to parse array JSON: %v", err)
	}

	if len(pets) == 0 {
		t.Errorf("expected non-empty array of pets")
	}

	firstPet := pets[0]
	if firstPet["id"] == nil || firstPet["id"] == "" {
		t.Errorf("expected pet to have an id")
	}
}

func init() {
	if _, err := os.Stat("../../testdata/petstore.yaml"); err != nil {
		if _, errRoot := os.Stat("testdata/petstore.yaml"); errRoot == nil {
			_ = os.Chdir("../..")
		}
	}
}
