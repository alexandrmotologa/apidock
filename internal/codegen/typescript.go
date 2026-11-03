package codegen

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// GenerateTypeScriptFetch builds modern TypeScript Fetch API code.
func GenerateTypeScriptFetch(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("const url = %q;\n", targetURL))
	sb.WriteString("const options: RequestInit = {\n")
	sb.WriteString(fmt.Sprintf("  method: %q,\n", method))

	if len(headers) > 0 {
		sb.WriteString("  headers: {\n")
		var keys []string
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("    %q: %q,\n", k, headers[k]))
		}
		sb.WriteString("  },\n")
	}

	if body != "" && method != "GET" && method != "HEAD" {
		// If valid JSON, stringify properly
		var js any
		if err := json.Unmarshal([]byte(body), &js); err == nil {
			compactJSON, _ := json.Marshal(js)
			sb.WriteString(fmt.Sprintf("  body: JSON.stringify(%s),\n", string(compactJSON)))
		} else {
			sb.WriteString(fmt.Sprintf("  body: %q,\n", body))
		}
	}

	sb.WriteString("};\n\n")
	sb.WriteString("try {\n")
	sb.WriteString("  const response = await fetch(url, options);\n")
	sb.WriteString("  if (!response.ok) {\n")
	sb.WriteString("    throw new Error(`HTTP error! status: ${response.status}`);\n")
	sb.WriteString("  }\n")
	sb.WriteString("  const data = await response.json();\n")
	sb.WriteString("  console.log(data);\n")
	sb.WriteString("} catch (error) {\n")
	sb.WriteString("  console.error('Request failed:', error);\n")
	sb.WriteString("}\n")

	return sb.String()
}

// GenerateTypeScriptAxios builds Axios request code.
func GenerateTypeScriptAxios(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToLower(strings.TrimSpace(method))
	if method == "" {
		method = "get"
	}

	var sb strings.Builder
	sb.WriteString("import axios from 'axios';\n\n")
	sb.WriteString("try {\n")
	sb.WriteString("  const response = await axios({\n")
	sb.WriteString(fmt.Sprintf("    method: %q,\n", method))
	sb.WriteString(fmt.Sprintf("    url: %q,\n", targetURL))

	if len(headers) > 0 {
		sb.WriteString("    headers: {\n")
		var keys []string
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("      %q: %q,\n", k, headers[k]))
		}
		sb.WriteString("    },\n")
	}

	if body != "" && method != "get" && method != "head" {
		var js any
		if err := json.Unmarshal([]byte(body), &js); err == nil {
			indentJSON, _ := json.MarshalIndent(js, "    ", "  ")
			sb.WriteString(fmt.Sprintf("    data: %s,\n", string(indentJSON)))
		} else {
			sb.WriteString(fmt.Sprintf("    data: %q,\n", body))
		}
	}

	sb.WriteString("  });\n")
	sb.WriteString("  console.log(response.data);\n")
	sb.WriteString("} catch (error) {\n")
	sb.WriteString("  console.error('Axios request error:', error);\n")
	sb.WriteString("}\n")

	return sb.String()
}
