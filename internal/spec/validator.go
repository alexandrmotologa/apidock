package spec

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var pathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)

// Diagnostic represents an issue found during OpenAPI specification validation.
type Diagnostic struct {
	Severity string `json:"severity"` // "ERROR" or "WARNING"
	Path     string `json:"path"`     // e.g. "paths./pets.{petId}.get"
	Message  string `json:"message"`
}

// Validator validates a normalized Spec for common structural issues,
// missing parameters, and broken references.
type Validator struct{}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate analyzes the Spec and returns a slice of Diagnostics.
func (v *Validator) Validate(spec *Spec) []Diagnostic {
	var diagnostics []Diagnostic

	// 1. Info validation
	if strings.TrimSpace(spec.Info.Title) == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "ERROR",
			Path:     "info.title",
			Message:  "API specification must define a non-empty info.title",
		})
	}
	if strings.TrimSpace(spec.Info.Version) == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "WARNING",
			Path:     "info.version",
			Message:  "API specification should declare an info.version",
		})
	}

	// 2. Server validation
	for i, srv := range spec.Servers {
		if strings.TrimSpace(srv.URL) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "ERROR",
				Path:     fmt.Sprintf("servers[%d].url", i),
				Message:  "Server URL cannot be empty",
			})
			continue
		}
		// If not a relative URL or containing variables, validate URL syntax
		if !strings.HasPrefix(srv.URL, "/") && !strings.Contains(srv.URL, "{") {
			if _, err := url.Parse(srv.URL); err != nil {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "WARNING",
					Path:     fmt.Sprintf("servers[%d].url", i),
					Message:  fmt.Sprintf("Invalid server URL format: %s", srv.URL),
				})
			}
		}
	}

	// 3. Endpoints validation
	if len(spec.Endpoints) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "WARNING",
			Path:     "paths",
			Message:  "No paths or operations found in specification",
		})
	}

	for _, ep := range spec.Endpoints {
		epPathKey := fmt.Sprintf("%s %s", ep.Method, ep.Path)

		// Path must start with /
		if !strings.HasPrefix(ep.Path, "/") {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "ERROR",
				Path:     epPathKey,
				Message:  "Path must begin with a forward slash '/'",
			})
		}

		// Check path parameters matching
		matches := pathParamRegex.FindAllStringSubmatch(ep.Path, -1)
		declaredParams := make(map[string]bool)
		for _, p := range ep.Parameters {
			if p.In == "path" {
				declaredParams[p.Name] = true
				if !p.Required {
					diagnostics = append(diagnostics, Diagnostic{
						Severity: "ERROR",
						Path:     fmt.Sprintf("%s.parameters.%s", epPathKey, p.Name),
						Message:  "Path parameters must have 'required: true'",
					})
				}
			}
		}

		for _, m := range matches {
			if len(m) > 1 {
				paramName := m[1]
				if !declaredParams[paramName] {
					diagnostics = append(diagnostics, Diagnostic{
						Severity: "ERROR",
						Path:     epPathKey,
						Message:  fmt.Sprintf("Path contains template '{%s}' but no corresponding path parameter is defined", paramName),
					})
				}
			}
		}

		// Check responses
		if len(ep.Responses) == 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "WARNING",
				Path:     fmt.Sprintf("%s.responses", epPathKey),
				Message:  "Operation defines no response codes",
			})
		}
	}

	return diagnostics
}
