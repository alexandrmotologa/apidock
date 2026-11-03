package diff

import (
	"fmt"
	"strings"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

// ChangeType classifies the impact of an OpenAPI difference.
type ChangeType string

const (
	ChangeBreaking    ChangeType = "BREAKING"
	ChangeNonBreaking ChangeType = "NON_BREAKING"
	ChangeInfo        ChangeType = "INFO"
)

// Change represents a discrete difference between two specification versions.
type Change struct {
	Type     ChangeType `json:"type"`
	Category string     `json:"category"` // "ENDPOINT", "PARAMETER", "RESPONSE", "SCHEMA"
	Path     string     `json:"path"`
	Message  string     `json:"message"`
}

// Report aggregates all differences between old and new specifications.
type Report struct {
	OldTitle         string   `json:"oldTitle"`
	NewTitle         string   `json:"newTitle"`
	OldVersion       string   `json:"oldVersion"`
	NewVersion       string   `json:"newVersion"`
	BreakingCount    int      `json:"breakingCount"`
	NonBreakingCount int      `json:"nonBreakingCount"`
	Changes          []Change `json:"changes"`
}

// Compare analyzes differences between oldSpec and newSpec.
func Compare(oldSpec, newSpec *spec.Spec) *Report {
	report := &Report{
		OldTitle:   oldSpec.Info.Title,
		NewTitle:   newSpec.Info.Title,
		OldVersion: oldSpec.Info.Version,
		NewVersion: newSpec.Info.Version,
		Changes:    make([]Change, 0),
	}

	// 1. Version change check
	if oldSpec.Info.Version != newSpec.Info.Version {
		report.Changes = append(report.Changes, Change{
			Type:     ChangeInfo,
			Category: "VERSION",
			Path:     "info.version",
			Message:  fmt.Sprintf("Version bumped from %s to %s", oldSpec.Info.Version, newSpec.Info.Version),
		})
	}

	// Index endpoints by METHOD + PATH
	oldEndpoints := make(map[string]spec.Endpoint)
	for _, ep := range oldSpec.Endpoints {
		key := fmt.Sprintf("%s %s", strings.ToUpper(ep.Method), ep.Path)
		oldEndpoints[key] = ep
	}

	newEndpoints := make(map[string]spec.Endpoint)
	for _, ep := range newSpec.Endpoints {
		key := fmt.Sprintf("%s %s", strings.ToUpper(ep.Method), ep.Path)
		newEndpoints[key] = ep
	}

	// Check removed endpoints (BREAKING)
	for key, ep := range oldEndpoints {
		if _, exists := newEndpoints[key]; !exists {
			report.Changes = append(report.Changes, Change{
				Type:     ChangeBreaking,
				Category: "ENDPOINT",
				Path:     key,
				Message:  fmt.Sprintf("Endpoint %s was removed", key),
			})
		} else {
			// Compare endpoint details
			newEp := newEndpoints[key]
			compareEndpointDetails(ep, newEp, key, report)
		}
	}

	// Check added endpoints (NON_BREAKING)
	for key := range newEndpoints {
		if _, exists := oldEndpoints[key]; !exists {
			report.Changes = append(report.Changes, Change{
				Type:     ChangeNonBreaking,
				Category: "ENDPOINT",
				Path:     key,
				Message:  fmt.Sprintf("Endpoint %s was added", key),
			})
		}
	}

	// Count breakages
	for _, c := range report.Changes {
		if c.Type == ChangeBreaking {
			report.BreakingCount++
		} else if c.Type == ChangeNonBreaking {
			report.NonBreakingCount++
		}
	}

	return report
}

func compareEndpointDetails(oldEp, newEp spec.Endpoint, epKey string, report *Report) {
	// 1. Compare Parameters
	oldParams := make(map[string]spec.Parameter)
	for _, p := range oldEp.Parameters {
		oldParams[p.In+":"+p.Name] = p
	}

	newParams := make(map[string]spec.Parameter)
	for _, p := range newEp.Parameters {
		newParams[p.In+":"+p.Name] = p
	}

	// Check new required parameters (BREAKING)
	for k, newP := range newParams {
		if oldP, exists := oldParams[k]; !exists {
			if newP.Required {
				report.Changes = append(report.Changes, Change{
					Type:     ChangeBreaking,
					Category: "PARAMETER",
					Path:     fmt.Sprintf("%s (%s %s)", epKey, newP.In, newP.Name),
					Message:  fmt.Sprintf("New required parameter '%s' added", newP.Name),
				})
			} else {
				report.Changes = append(report.Changes, Change{
					Type:     ChangeNonBreaking,
					Category: "PARAMETER",
					Path:     fmt.Sprintf("%s (%s %s)", epKey, newP.In, newP.Name),
					Message:  fmt.Sprintf("New optional parameter '%s' added", newP.Name),
				})
			}
		} else {
			// Parameter became required
			if !oldP.Required && newP.Required {
				report.Changes = append(report.Changes, Change{
					Type:     ChangeBreaking,
					Category: "PARAMETER",
					Path:     fmt.Sprintf("%s (%s %s)", epKey, newP.In, newP.Name),
					Message:  fmt.Sprintf("Parameter '%s' changed from optional to required", newP.Name),
				})
			}
		}
	}

	// Check removed parameters
	for k, oldP := range oldParams {
		if _, exists := newParams[k]; !exists {
			changeType := ChangeNonBreaking
			if oldP.Required {
				changeType = ChangeBreaking
			}
			report.Changes = append(report.Changes, Change{
				Type:     changeType,
				Category: "PARAMETER",
				Path:     fmt.Sprintf("%s (%s %s)", epKey, oldP.In, oldP.Name),
				Message:  fmt.Sprintf("Parameter '%s' was removed", oldP.Name),
			})
		}
	}

	// 2. Compare Responses
	for code := range oldEp.Responses {
		if _, exists := newEp.Responses[code]; !exists {
			report.Changes = append(report.Changes, Change{
				Type:     ChangeBreaking,
				Category: "RESPONSE",
				Path:     fmt.Sprintf("%s [HTTP %s]", epKey, code),
				Message:  fmt.Sprintf("Response code %s was removed", code),
			})
		}
	}

	for code := range newEp.Responses {
		if _, exists := oldEp.Responses[code]; !exists {
			report.Changes = append(report.Changes, Change{
				Type:     ChangeNonBreaking,
				Category: "RESPONSE",
				Path:     fmt.Sprintf("%s [HTTP %s]", epKey, code),
				Message:  fmt.Sprintf("Response code %s was added", code),
			})
		}
	}
}
