package codegen

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateCurl builds a copy-pasteable cURL command.
func GenerateCurl(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("curl -X %s %q", method, targetURL))

	// Sort headers for deterministic output
	var headerKeys []string
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)

	for _, k := range headerKeys {
		v := headers[k]
		sb.WriteString(fmt.Sprintf(" \\\n  -H %q", fmt.Sprintf("%s: %s", k, v)))
	}

	if body != "" && method != "GET" && method != "HEAD" {
		escapedBody := escapeForShell(body)
		sb.WriteString(fmt.Sprintf(" \\\n  -d %s", escapedBody))
	}

	return sb.String()
}

func escapeForShell(s string) string {
	// Single quote escaping for bash/zsh/sh
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
