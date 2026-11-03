package codegen

import (
	"strings"
	"testing"
)

func TestGenerateAllSnippets(t *testing.T) {
	method := "POST"
	targetURL := "https://api.example.com/v1/pets"
	headers := map[string]string{
		"Authorization": "Bearer sec_test_123",
		"Content-Type":  "application/json",
	}
	body := `{"name":"Buddy","tag":"dog"}`

	snippets := GenerateAll(method, targetURL, headers, body)

	// Verify cURL
	if !strings.Contains(snippets.Curl, "curl -X POST") {
		t.Errorf("expected curl to have POST, got %s", snippets.Curl)
	}
	if !strings.Contains(snippets.Curl, "-H \"Authorization: Bearer sec_test_123\"") {
		t.Errorf("expected curl to contain authorization header, got %s", snippets.Curl)
	}
	if !strings.Contains(snippets.Curl, "-d '{\"name\":\"Buddy\",\"tag\":\"dog\"}'") {
		t.Errorf("expected curl to contain body, got %s", snippets.Curl)
	}

	// Verify TypeScript Fetch
	if !strings.Contains(snippets.TypeScriptFetch, "fetch(url, options)") {
		t.Errorf("expected ts fetch to contain fetch call")
	}
	if !strings.Contains(snippets.TypeScriptFetch, "\"Authorization\": \"Bearer sec_test_123\"") {
		t.Errorf("expected ts fetch to have auth header")
	}

	// Verify TypeScript Axios
	if !strings.Contains(snippets.TypeScriptAxios, "axios({") {
		t.Errorf("expected axios snippet")
	}

	// Verify Python Requests
	if !snippetsContains(snippets.PythonRequests, "requests.post(url") {
		t.Errorf("expected python requests post call")
	}

	// Verify Python HTTPX
	if !snippetsContains(snippets.PythonHTTPX, "client.post(url") {
		t.Errorf("expected python httpx post call")
	}

	// Verify Go net/http
	if !snippetsContains(snippets.GoNetHTTP, "http.NewRequest(\"POST\"") {
		t.Errorf("expected go net/http NewRequest call")
	}
}

func snippetsContains(s, sub string) bool {
	return strings.Contains(s, sub)
}
