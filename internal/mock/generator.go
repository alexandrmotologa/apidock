package mock

import (
	"crypto/rand"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

// Generator produces synthetic and example-driven mock payloads.
type Generator struct {
	spec *spec.Spec
}

// NewGenerator creates a new schema-driven mock data generator.
func NewGenerator(s *spec.Spec) *Generator {
	return &Generator{spec: s}
}

// GenerateResponse determines the response status, headers, and body for a given endpoint and request directives.
func (g *Generator) GenerateResponse(
	ep *spec.Endpoint,
	preferredStatus string,
	preferredExample string,
) (int, map[string]string, any) {
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"X-Powered-By": "APIDock-Mock-Engine",
	}

	if ep == nil || len(ep.Responses) == 0 {
		return 200, headers, map[string]any{"message": "OK (Mock endpoint with no declared responses)"}
	}

	// 1. Choose status code
	statusCodeStr, respDef := g.selectResponse(ep, preferredStatus)
	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		statusCode = 200
	}

	if statusCode == 204 || respDef == nil || len(respDef.Content) == 0 {
		return statusCode, headers, nil
	}

	// 2. Select media type (prefer application/json)
	mt, contentType := g.selectMediaType(respDef)
	headers["Content-Type"] = contentType

	// 3. Resolve example or generate synthetic payload
	data := g.resolvePayload(mt, preferredExample)
	return statusCode, headers, data
}

func (g *Generator) selectResponse(ep *spec.Endpoint, preferred string) (string, *spec.Response) {
	// If explicitly preferred status code exists
	if preferred != "" {
		if resp, ok := ep.Responses[preferred]; ok {
			return preferred, &resp
		}
	}

	// Collect and sort all codes
	var codes []string
	for code := range ep.Responses {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	// Prefer 200, 201, 204 or any 2xx
	for _, code := range codes {
		if strings.HasPrefix(code, "2") {
			resp := ep.Responses[code]
			return code, &resp
		}
	}

	// Fallback to first available numeric code
	for _, code := range codes {
		if code != "default" {
			resp := ep.Responses[code]
			return code, &resp
		}
	}

	// Fallback to "default" if defined
	if resp, ok := ep.Responses["default"]; ok {
		return "200", &resp
	}

	return "200", nil
}

func (g *Generator) selectMediaType(resp *spec.Response) (spec.MediaType, string) {
	if mt, ok := resp.Content["application/json"]; ok {
		return mt, "application/json; charset=utf-8"
	}

	// Otherwise pick the first content type defined
	for ct, mt := range resp.Content {
		return mt, ct
	}

	return spec.MediaType{}, "application/json; charset=utf-8"
}

func (g *Generator) resolvePayload(mt spec.MediaType, preferredExample string) any {
	// 1. Preferred named example
	if preferredExample != "" && mt.Examples != nil {
		if ex, ok := mt.Examples[preferredExample]; ok {
			return ex.Value
		}
	}

	// 2. MediaType-level single example
	if mt.Example != nil {
		return mt.Example
	}

	// 3. First named example if available
	if len(mt.Examples) > 0 {
		for _, ex := range mt.Examples {
			return ex.Value
		}
	}

	// 4. Schema-level example
	if mt.Schema != nil {
		if mt.Schema.Example != nil {
			return mt.Schema.Example
		}
		if len(mt.Schema.Examples) > 0 {
			return mt.Schema.Examples[0]
		}
		// 5. Generate recursively from schema definition
		return g.generateFromSchema(mt.Schema, 0)
	}

	return map[string]any{"status": "ok"}
}

func (g *Generator) generateFromSchema(s *spec.Schema, depth int) any {
	if s == nil || depth > 10 {
		return nil
	}

	// Resolve schema $ref if unexpanded
	if s.Ref != "" && len(s.Properties) == 0 && (s.Type == "" || s.Type == "object") && g.spec != nil {
		parts := strings.Split(s.Ref, "/")
		name := parts[len(parts)-1]
		if target, ok := g.spec.Components.Schemas[name]; ok && target != nil && target != s {
			return g.generateFromSchema(target, depth+1)
		}
	}

	// Handle polymorphism (oneOf, anyOf, allOf)
	if len(s.OneOf) > 0 {
		return g.generateFromSchema(s.OneOf[0], depth+1)
	}
	if len(s.AnyOf) > 0 {
		return g.generateFromSchema(s.AnyOf[0], depth+1)
	}
	if len(s.AllOf) > 0 {
		merged := make(map[string]any)
		for _, part := range s.AllOf {
			sub := g.generateFromSchema(part, depth+1)
			if m, ok := sub.(map[string]any); ok {
				for k, v := range m {
					merged[k] = v
				}
			}
		}
		return merged
	}

	// Check enum values
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}

	// Default value if set
	if s.Default != nil {
		return s.Default
	}

	// Example value if set
	if s.Example != nil {
		return s.Example
	}

	schemaType := s.Type
	if schemaType == "" && len(s.Types) > 0 {
		schemaType = s.Types[0]
	}

	// Heuristics based on schema type and format
	switch schemaType {
	case "string":
		return g.generateString(s)
	case "integer":
		return g.generateInteger(s)
	case "number":
		return g.generateNumber(s)
	case "boolean":
		return true
	case "array":
		return g.generateArray(s, depth)
	case "object", "":
		return g.generateObject(s, depth)
	default:
		return "mock_value"
	}
}

func (g *Generator) generateString(s *spec.Schema) string {
	switch s.Format {
	case "uuid":
		return randomUUID()
	case "email":
		return "alex.motologa@example.com"
	case "date-time":
		return time.Now().UTC().Format(time.RFC3339)
	case "date":
		return time.Now().UTC().Format("2006-01-02")
	case "time":
		return time.Now().UTC().Format("15:04:05")
	case "uri", "url":
		return "https://api.example.com/v1/resource"
	case "ipv4":
		return "192.0.2.1"
	case "ipv6":
		return "2001:db8::1"
	case "hostname":
		return "api.example.com"
	case "byte":
		return "ZXhhbXBsZQ=="
	case "binary":
		return "sample-binary-content"
	default:
		if s.Title != "" {
			return strings.ToLower(s.Title) + "_sample"
		}
		return "string_sample"
	}
}

func (g *Generator) generateInteger(s *spec.Schema) int64 {
	if s.Minimum != nil {
		return int64(*s.Minimum)
	}
	if s.Maximum != nil && *s.Maximum < 100 {
		return int64(*s.Maximum)
	}
	return 42
}

func (g *Generator) generateNumber(s *spec.Schema) float64 {
	if s.Minimum != nil {
		return *s.Minimum
	}
	return 29.99
}

func (g *Generator) generateArray(s *spec.Schema, depth int) []any {
	if s.Items == nil {
		return []any{}
	}
	// Produce 1 to 2 items
	count := 2
	if s.MinItems != nil && *s.MinItems > 0 {
		count = int(*s.MinItems)
		if count > 5 {
			count = 5
		}
	}
	items := make([]any, count)
	for i := 0; i < count; i++ {
		items[i] = g.generateFromSchema(s.Items, depth+1)
	}
	return items
}

func (g *Generator) generateObject(s *spec.Schema, depth int) map[string]any {
	obj := make(map[string]any)

	// Properties
	if len(s.Properties) > 0 {
		for propName, propSchema := range s.Properties {
			val := g.generateFromSchema(propSchema, depth+1)
			obj[propName] = val
		}
		return obj
	}

	// AdditionalProperties
	if s.AdditionalProperties != nil {
		if addSchema, ok := s.AdditionalProperties.(*spec.Schema); ok && addSchema != nil {
			obj["key1"] = g.generateFromSchema(addSchema, depth+1)
			obj["key2"] = g.generateFromSchema(addSchema, depth+1)
			return obj
		}
		obj["mockKey"] = "mockValue"
		return obj
	}

	return obj
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant RFC 4122
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
