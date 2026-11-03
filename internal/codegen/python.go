package codegen

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GeneratePythonRequests generates Python code using the `requests` library.
func GeneratePythonRequests(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "" {
		method = "get"
	}

	var sb strings.Builder
	sb.WriteString("import requests\n\n")
	sb.WriteString(fmt.Sprintf("url = %q\n", targetURL))

	if len(headers) > 0 {
		sb.WriteString("headers = {\n")
		var keys []string
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("    %q: %q,\n", k, headers[k]))
		}
		sb.WriteString("}\n")
	} else {
		sb.WriteString("headers = {}\n")
	}

	hasBody := body != "" && method != "get" && method != "head"
	if hasBody {
		var js any
		if err := json.Unmarshal([]byte(body), &js); err == nil {
			compactJSON, _ := json.Marshal(js)
			sb.WriteString(fmt.Sprintf("payload = %s\n", string(compactJSON)))
		} else {
			sb.WriteString(fmt.Sprintf("payload = %q\n", body))
		}
		sb.WriteString(fmt.Sprintf("\nresponse = requests.%s(url, headers=headers, json=payload)\n", method))
	} else {
		sb.WriteString(fmt.Sprintf("\nresponse = requests.%s(url, headers=headers)\n", method))
	}

	sb.WriteString("print(f'Status: {response.status_code}')\n")
	sb.WriteString("try:\n")
	sb.WriteString("    print(response.json())\n")
	sb.WriteString("except Exception:\n")
	sb.WriteString("    print(response.text)\n")

	return sb.String()
}

// GeneratePythonHTTPX generates Python code using `httpx`.
func GeneratePythonHTTPX(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "" {
		method = "get"
	}

	var sb strings.Builder
	sb.WriteString("import httpx\n\n")
	sb.WriteString(fmt.Sprintf("url = %q\n", targetURL))

	if len(headers) > 0 {
		sb.WriteString("headers = {\n")
		var keys []string
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("    %q: %q,\n", k, headers[k]))
		}
		sb.WriteString("}\n")
	} else {
		sb.WriteString("headers = {}\n")
	}

	hasBody := body != "" && method != "get" && method != "head"
	if hasBody {
		var js any
		if err := json.Unmarshal([]byte(body), &js); err == nil {
			compactJSON, _ := json.Marshal(js)
			sb.WriteString(fmt.Sprintf("payload = %s\n", string(compactJSON)))
		} else {
			sb.WriteString(fmt.Sprintf("payload = %q\n", body))
		}
		sb.WriteString(fmt.Sprintf("\nwith httpx.Client() as client:\n    response = client.%s(url, headers=headers, json=payload)\n", method))
	} else {
		sb.WriteString(fmt.Sprintf("\nwith httpx.Client() as client:\n    response = client.%s(url, headers=headers)\n", method))
	}

	sb.WriteString("    print(f'Status: {response.status_code}')\n")
	sb.WriteString("    print(response.json() if 'application/json' in response.headers.get('content-type', '') else response.text)\n")

	return sb.String()
}
