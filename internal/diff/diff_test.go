package diff_test

import (
	"testing"

	"github.com/alexandrmotologa/apidock/internal/diff"
	"github.com/alexandrmotologa/apidock/internal/spec"
)

func TestCompareSpecs(t *testing.T) {
	oldSpec := &spec.Spec{
		Info: spec.Info{
			Title:   "Petstore API",
			Version: "1.0.0",
		},
		Endpoints: []spec.Endpoint{
			{
				Method: "get",
				Path:   "/pets",
				Parameters: []spec.Parameter{
					{Name: "limit", In: "query", Required: false},
					{Name: "old_token", In: "header", Required: true},
				},
				Responses: map[string]spec.Response{
					"200": {Description: "OK"},
					"404": {Description: "Not found"},
				},
			},
			{
				Method: "delete",
				Path:   "/pets/{id}",
			},
		},
	}

	newSpec := &spec.Spec{
		Info: spec.Info{
			Title:   "Petstore API",
			Version: "2.0.0",
		},
		Endpoints: []spec.Endpoint{
			{
				Method: "get",
				Path:   "/pets",
				Parameters: []spec.Parameter{
					{Name: "limit", In: "query", Required: true}, // Changed to required (BREAKING)
					{Name: "filter", In: "query", Required: true}, // New required param (BREAKING)
					{Name: "sort", In: "query", Required: false},  // New optional param (NON_BREAKING)
				},
				Responses: map[string]spec.Response{
					"200": {Description: "OK"},
					"201": {Description: "Created"}, // Added response code (NON_BREAKING)
					// "404" removed (BREAKING)
				},
			},
			{
				Method: "post",
				Path:   "/pets", // Added endpoint (NON_BREAKING)
			},
			// "delete /pets/{id}" removed (BREAKING)
		},
	}

	report := diff.Compare(oldSpec, newSpec)

	if report.OldVersion != "1.0.0" || report.NewVersion != "2.0.0" {
		t.Fatalf("expected version 1.0.0 -> 2.0.0, got %s -> %s", report.OldVersion, report.NewVersion)
	}

	if report.BreakingCount <= 0 {
		t.Fatalf("expected breaking changes count > 0, got %d", report.BreakingCount)
	}

	if report.NonBreakingCount <= 0 {
		t.Fatalf("expected non-breaking changes count > 0, got %d", report.NonBreakingCount)
	}

	// Verify specific breaking items
	foundRemovedEndpoint := false
	foundRemovedResponse := false
	foundNewRequiredParam := false

	for _, c := range report.Changes {
		if c.Type == diff.ChangeBreaking {
			if c.Category == "ENDPOINT" && c.Path == "DELETE /pets/{id}" {
				foundRemovedEndpoint = true
			}
			if c.Category == "RESPONSE" && c.Path == "GET /pets [HTTP 404]" {
				foundRemovedResponse = true
			}
			if c.Category == "PARAMETER" && c.Message == "New required parameter 'filter' added" {
				foundNewRequiredParam = true
			}
		}
	}

	if !foundRemovedEndpoint {
		t.Errorf("expected breaking change for removed endpoint DELETE /pets/{id}")
	}
	if !foundRemovedResponse {
		t.Errorf("expected breaking change for removed HTTP 404 response")
	}
	if !foundNewRequiredParam {
		t.Errorf("expected breaking change for new required parameter 'filter'")
	}
}
