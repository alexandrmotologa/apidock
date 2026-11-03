package export_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/alexandrmotologa/apidock/internal/export"
	"github.com/alexandrmotologa/apidock/internal/spec"
)

func createSampleSpec() *spec.Spec {
	return &spec.Spec{
		OpenAPI: "3.1.0",
		Info: spec.Info{
			Title:       "Test Store API",
			Description: "API for testing exporters",
			Version:     "1.0.0",
		},
		Servers: []spec.Server{
			{URL: "https://api.teststore.com/v1"},
		},
		Endpoints: []spec.Endpoint{
			{
				Method:      "get",
				Path:        "/products",
				Summary:     "List all products",
				Description: "Returns all products in catalog",
				Tags:        []string{"Catalog"},
				Parameters: []spec.Parameter{
					{Name: "limit", In: "query", Example: 20},
				},
				Responses: map[string]spec.Response{
					"200": {Description: "Successful response"},
				},
			},
			{
				Method:      "post",
				Path:        "/products",
				Summary:     "Create product",
				Description: "Adds a new product",
				Tags:        []string{"Catalog"},
				RequestBody: &spec.RequestBody{
					Content: map[string]spec.MediaType{
						"application/json": {
							Example: map[string]any{"name": "Widget", "price": 49.99},
						},
					},
				},
				Responses: map[string]spec.Response{
					"201": {Description: "Created"},
				},
			},
		},
	}
}

func TestPostmanExport(t *testing.T) {
	s := createSampleSpec()
	data, err := export.ToPostmanJSON(s)
	if err != nil {
		t.Fatalf("unexpected error exporting to Postman: %v", err)
	}

	var col export.PostmanCollection
	if err := json.Unmarshal(data, &col); err != nil {
		t.Fatalf("failed to parse generated Postman JSON: %v", err)
	}

	if col.Info.Name != "Test Store API" {
		t.Errorf("expected collection name 'Test Store API', got '%s'", col.Info.Name)
	}

	if len(col.Item) == 0 {
		t.Fatalf("expected folders or items in postman collection")
	}

	// Verify Catalog folder
	catalogFolder := col.Item[0]
	if catalogFolder.Name != "Catalog" {
		t.Errorf("expected folder 'Catalog', got '%s'", catalogFolder.Name)
	}

	if len(catalogFolder.Item) != 2 {
		t.Errorf("expected 2 endpoints in Catalog folder, got %d", len(catalogFolder.Item))
	}
}

func TestBrunoExport(t *testing.T) {
	s := createSampleSpec()
	tempDir, err := os.MkdirTemp("", "apidock-bruno-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := export.ExportBrunoToDir(s, tempDir); err != nil {
		t.Fatalf("unexpected error exporting to Bruno: %v", err)
	}

	// Check bruno.json
	metaPath := filepath.Join(tempDir, "bruno.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatalf("bruno.json was not created")
	}

	// Check environments/Local.bru
	envPath := filepath.Join(tempDir, "environments", "Local.bru")
	envBytes, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read Local.bru: %v", err)
	}
	if !strings.Contains(string(envBytes), "https://api.teststore.com/v1") {
		t.Errorf("expected Local.bru to contain server URL, got: %s", string(envBytes))
	}

	// Check catalog folder
	catalogDir := filepath.Join(tempDir, "catalog")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		t.Fatalf("failed to read catalog dir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 .bru files in catalog dir, got %d", len(entries))
	}
}

func TestSingleFileHTMLExport(t *testing.T) {
	s := createSampleSpec()

	mockFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><head><title>APIDock</title><link rel="stylesheet" crossorigin href="/assets/style.css"><script type="module" crossorigin src="/assets/main.js"></script></head><body><div id="root"></div></body></html>`),
		},
		"assets/style.css": &fstest.MapFile{
			Data: []byte(`body { background: #000; color: #fff; }`),
		},
		"assets/main.js": &fstest.MapFile{
			Data: []byte(`console.log("APIDock standalone booted");`),
		},
	}

	htmlBytes, err := export.BuildSingleFileHTML(s, mockFS)
	if err != nil {
		t.Fatalf("failed to build single-file HTML: %v", err)
	}

	htmlStr := string(htmlBytes)

	// Ensure external links were replaced with inlined tags
	if strings.Contains(htmlStr, `href="/assets/style.css"`) {
		t.Errorf("expected css link to be inlined and removed")
	}
	if !strings.Contains(htmlStr, `<style>`) || !strings.Contains(htmlStr, `background: #000;`) {
		t.Errorf("expected inlined CSS style tag")
	}

	if strings.Contains(htmlStr, `src="/assets/main.js"`) {
		t.Errorf("expected script src to be inlined and removed")
	}
	if !strings.Contains(htmlStr, `<script type="module">`) || !strings.Contains(htmlStr, `APIDock standalone booted`) {
		t.Errorf("expected inlined JS script tag")
	}

	// Ensure spec JSON is injected
	if !strings.Contains(htmlStr, `id="apidock-spec"`) || !strings.Contains(htmlStr, `Test Store API`) {
		t.Errorf("expected embedded spec JSON payload in document")
	}

	// Ensure title is updated
	if !strings.Contains(htmlStr, `<title>Test Store API - APIDock</title>`) {
		t.Errorf("expected updated title tag, got: %s", htmlStr)
	}
}
