package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

// BrunoMeta contains Bruno collection metadata for bruno.json.
type BrunoMeta struct {
	Version string   `json:"version"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Ignore  []string `json:"ignore"`
}

// ExportBrunoToDir exports the OpenAPI specification as a Bruno collection structure.
func ExportBrunoToDir(s *spec.Spec, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create bruno directory %s: %w", targetDir, err)
	}

	// 1. Create bruno.json
	meta := BrunoMeta{
		Version: "1",
		Name:    s.Info.Title,
		Type:    "collection",
		Ignore:  []string{"node_modules", ".git"},
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "bruno.json"), metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to write bruno.json: %w", err)
	}

	// 2. Create environments/Local.bru
	envDir := filepath.Join(targetDir, "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return err
	}
	baseUrl := "http://localhost:8080"
	if len(s.Servers) > 0 {
		baseUrl = s.Servers[0].URL
	}
	envContent := fmt.Sprintf("vars {\n  baseUrl: %s\n}\n", baseUrl)
	if err := os.WriteFile(filepath.Join(envDir, "Local.bru"), []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write Local.bru: %w", err)
	}

	// 3. Group endpoints by tag
	tagGroups := make(map[string][]spec.Endpoint)
	for _, ep := range s.Endpoints {
		tag := "General"
		if len(ep.Tags) > 0 {
			tag = ep.Tags[0]
		}
		tagGroups[tag] = append(tagGroups[tag], ep)
	}

	for tag, endpoints := range tagGroups {
		folderName := sanitizeFileName(tag)
		folderPath := filepath.Join(targetDir, folderName)
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return err
		}

		for i, ep := range endpoints {
			filename := generateBruFileName(ep, i+1)
			bruFilePath := filepath.Join(folderPath, filename)
			bruContent := GenerateBruContent(ep, i+1)
			if err := os.WriteFile(bruFilePath, []byte(bruContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", bruFilePath, err)
			}
		}
	}

	return nil
}

// GenerateBruContent converts a single endpoint into .bru file DSL syntax.
func GenerateBruContent(ep spec.Endpoint, seq int) string {
	name := ep.Summary
	if name == "" {
		name = fmt.Sprintf("%s %s", strings.ToUpper(ep.Method), ep.Path)
	}

	var sb strings.Builder

	// Meta block
	sb.WriteString("meta {\n")
	sb.WriteString(fmt.Sprintf("  name: %s\n", name))
	sb.WriteString("  type: http\n")
	sb.WriteString(fmt.Sprintf("  seq: %d\n", seq))
	sb.WriteString("}\n\n")

	// Method block
	method := strings.ToLower(ep.Method)
	bodyType := "none"
	hasJsonBody := false
	var sampleJsonBody string

	if ep.RequestBody != nil && len(ep.RequestBody.Content) > 0 {
		for mime, mt := range ep.RequestBody.Content {
			if strings.Contains(mime, "json") {
				bodyType = "json"
				hasJsonBody = true
				if mt.Example != nil {
					if b, err := json.MarshalIndent(mt.Example, "", "  "); err == nil {
						sampleJsonBody = string(b)
					}
				}
				if sampleJsonBody == "" {
					sampleJsonBody = "{\n  \n}"
				}
				break
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%s {\n", method))
	sb.WriteString(fmt.Sprintf("  url: {{baseUrl}}%s\n", ep.Path))
	sb.WriteString(fmt.Sprintf("  body: %s\n", bodyType))
	sb.WriteString("  auth: none\n")
	sb.WriteString("}\n\n")

	// Headers
	sb.WriteString("headers {\n")
	sb.WriteString("  Accept: application/json\n")
	if hasJsonBody {
		sb.WriteString("  Content-Type: application/json\n")
	}
	for _, p := range ep.Parameters {
		if strings.ToLower(p.In) == "header" {
			sb.WriteString(fmt.Sprintf("  ~%s: \n", p.Name))
		}
	}
	sb.WriteString("}\n\n")

	// Query params if present
	var queryParams []spec.Parameter
	for _, p := range ep.Parameters {
		if strings.ToLower(p.In) == "query" {
			queryParams = append(queryParams, p)
		}
	}

	if len(queryParams) > 0 {
		sb.WriteString("params:query {\n")
		for _, q := range queryParams {
			val := ""
			if q.Example != nil {
				val = fmt.Sprintf("%v", q.Example)
			}
			sb.WriteString(fmt.Sprintf("  %s: %s\n", q.Name, val))
		}
		sb.WriteString("}\n\n")
	}

	// Body content if json
	if hasJsonBody && sampleJsonBody != "" {
		sb.WriteString("body:json {\n")
		sb.WriteString("  " + strings.ReplaceAll(sampleJsonBody, "\n", "\n  ") + "\n")
		sb.WriteString("}\n\n")
	}

	// Docs block
	if ep.Description != "" {
		sb.WriteString("docs {\n")
		sb.WriteString(fmt.Sprintf("  %s\n", strings.ReplaceAll(ep.Description, "\n", "\n  ")))
		sb.WriteString("}\n")
	}

	return sb.String()
}

var nonAlphaRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

func sanitizeFileName(s string) string {
	res := nonAlphaRegex.ReplaceAllString(s, "_")
	res = strings.Trim(res, "_")
	if res == "" {
		return "general"
	}
	return strings.ToLower(res)
}

func generateBruFileName(ep spec.Endpoint, seq int) string {
	base := ep.OperationID
	if base == "" {
		cleanPath := strings.ReplaceAll(ep.Path, "/", "-")
		cleanPath = strings.ReplaceAll(cleanPath, "{", "")
		cleanPath = strings.ReplaceAll(cleanPath, "}", "")
		base = fmt.Sprintf("%s%s", ep.Method, cleanPath)
	}
	return fmt.Sprintf("%02d-%s.bru", seq, sanitizeFileName(base))
}
