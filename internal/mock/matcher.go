package mock

import (
	"regexp"
	"strings"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

// RouteMatch represents a matched endpoint and extracted path parameters.
type RouteMatch struct {
	Endpoint   *spec.Endpoint
	PathParams map[string]string
}

// RouteRule stores compiled regex and parameter names for an endpoint.
type RouteRule struct {
	Endpoint   spec.Endpoint
	Regex      *regexp.Regexp
	ParamNames []string
}

// Matcher performs path matching for mock requests against OpenAPI endpoints.
type Matcher struct {
	rules []RouteRule
}

// NewMatcher compiles endpoint paths into matching rules.
func NewMatcher(endpoints []spec.Endpoint) *Matcher {
	rules := make([]RouteRule, 0, len(endpoints))

	for _, ep := range endpoints {
		regexStr, paramNames := compilePathToRegex(ep.Path)
		re, err := regexp.Compile(regexStr)
		if err == nil {
			rules = append(rules, RouteRule{
				Endpoint:   ep,
				Regex:      re,
				ParamNames: paramNames,
			})
		}
	}

	return &Matcher{rules: rules}
}

// Match finds an endpoint matching the given HTTP method and request URL path.
func (m *Matcher) Match(method, path string) *RouteMatch {
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}
	method = strings.ToUpper(method)

	// First pass: exact method match
	for _, rule := range m.rules {
		if strings.ToUpper(rule.Endpoint.Method) != method {
			continue
		}

		matches := rule.Regex.FindStringSubmatch(cleanPath)
		if matches != nil {
			params := make(map[string]string)
			for i, name := range rule.ParamNames {
				if i+1 < len(matches) {
					params[name] = matches[i+1]
				}
			}
			epCopy := rule.Endpoint
			return &RouteMatch{
				Endpoint:   &epCopy,
				PathParams: params,
			}
		}
	}

	return nil
}

// compilePathToRegex turns `/pets/{petId}/orders/{id}` into `^/pets/([^/]+)/orders/([^/]+)$`
func compilePathToRegex(apiPath string) (string, []string) {
	cleanPath := strings.TrimRight(apiPath, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}

	var paramNames []string
	segments := strings.Split(cleanPath, "/")
	var regexSegments []string

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			paramName := strings.Trim(seg, "{}")
			paramNames = append(paramNames, paramName)
			regexSegments = append(regexSegments, `([^/]+)`)
		} else {
			regexSegments = append(regexSegments, regexp.QuoteMeta(seg))
		}
	}

	var pattern string
	if len(regexSegments) == 0 {
		pattern = `^/$`
	} else {
		pattern = `^/` + strings.Join(regexSegments, `/`) + `/?$`
	}

	return pattern, paramNames
}
