package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

func loadServerTestSpec(t *testing.T) (*spec.Spec, []byte) {
	t.Helper()
	p := spec.NewParser()
	path := "../../testdata/petstore.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		path = "testdata/petstore.yaml"
		data, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read test spec: %v", err)
		}
	}
	s, err := p.ParseBytes(data)
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}
	return s, data
}

func TestServerEndpoints(t *testing.T) {
	s, rawBytes := loadServerTestSpec(t)
	apiServer := NewAPIServer(Config{
		Port:      8080,
		SpecPath:  "testdata/petstore.yaml",
		Spec:      s,
		SpecBytes: rawBytes,
	})

	// 1. GET /api/spec
	req := httptest.NewRequest(http.MethodGet, "/api/spec", nil)
	rec := httptest.NewRecorder()
	apiServer.handleGetSpec(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/spec, got %d", rec.Code)
	}

	// 2. GET /api/validate
	req = httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	rec = httptest.NewRecorder()
	apiServer.handleValidate(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/validate, got %d", rec.Code)
	}
	var valResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &valResp)
	if valResp["valid"] != true {
		t.Errorf("expected valid to be true")
	}

	// 3. GET /api/status
	req = httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec = httptest.NewRecorder()
	apiServer.handleStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/status, got %d", rec.Code)
	}

	// 4. Static fallback
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	apiServer.handleStatic(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for static index, got %d", rec.Code)
	}

	// 5. POST /api/diff
	diffBody := `{"newSpec":"openapi: 3.1.0\ninfo:\n  title: Petstore 2\n  version: 2.0.0\npaths: {}"}`
	diffReq := httptest.NewRequest(http.MethodPost, "/api/diff", strings.NewReader(diffBody))
	diffRec := httptest.NewRecorder()
	apiServer.handleDiff(diffRec, diffReq)
	if diffRec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/diff, got %d (body: %s)", diffRec.Code, diffRec.Body.String())
	}

	// 6. GET /api/export?format=postman
	exportReq := httptest.NewRequest(http.MethodGet, "/api/export?format=postman", nil)
	exportRec := httptest.NewRecorder()
	apiServer.handleExport(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Errorf("expected 200 for /api/export postman, got %d", exportRec.Code)
	}
}

func init() {
	if _, err := os.Stat("../../testdata/petstore.yaml"); err != nil {
		if _, errRoot := os.Stat("testdata/petstore.yaml"); errRoot == nil {
			_ = os.Chdir("../..")
		}
	}
}
