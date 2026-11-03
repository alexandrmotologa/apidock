package spec

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	libyaml "github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Parser loads and parses OpenAPI 3.0 and 3.1 specifications.
type Parser struct {
	httpClient *http.Client
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Parse loads and normalizes an OpenAPI document from a file path or URL.
func (p *Parser) Parse(pathOrURL string) (*Spec, error) {
	data, err := p.loadBytes(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec from %s: %w", pathOrURL, err)
	}

	resolver := NewResolver("")
	doc, err := resolver.BuildDocument(data, pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAPI document: %w", err)
	}

	return p.parseDocument(doc)
}

// ParseBytes parses and normalizes OpenAPI specification bytes.
func (p *Parser) ParseBytes(data []byte) (*Spec, error) {
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAPI document: %w", err)
	}

	return p.parseDocument(doc)
}

func (p *Parser) parseDocument(doc libopenapi.Document) (*Spec, error) {
	v3Model, modelErr := doc.BuildV3Model()
	if v3Model == nil {
		return nil, fmt.Errorf("failed to build OpenAPI v3 model: %w", modelErr)
	}

	spec := &Spec{
		OpenAPI:    v3Model.Model.Version,
		Servers:    make([]Server, 0),
		Tags:       make([]Tag, 0),
		Endpoints:  make([]Endpoint, 0),
		Components: Components{
			Schemas:         make(map[string]*Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
		},
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	if modelErr != nil {
		spec.Warnings = append(spec.Warnings, modelErr.Error())
	}

	// 1. Info
	if v3Model.Model.Info != nil {
		info := v3Model.Model.Info
		spec.Info = Info{
			Title:          info.Title,
			Description:    info.Description,
			Version:        info.Version,
			TermsOfService: info.TermsOfService,
		}
		if info.Contact != nil {
			spec.Info.Contact = &Contact{
				Name:  info.Contact.Name,
				URL:   info.Contact.URL,
				Email: info.Contact.Email,
			}
		}
		if info.License != nil {
			spec.Info.License = &License{
				Name: info.License.Name,
				URL:  info.License.URL,
			}
		}
	}

	// 2. Servers
	if len(v3Model.Model.Servers) > 0 {
		for _, s := range v3Model.Model.Servers {
			srv := Server{
				URL:         s.URL,
				Description: s.Description,
				Variables:   make(map[string]ServerVariable),
			}
			if s.Variables != nil {
				for pair := s.Variables.First(); pair != nil; pair = pair.Next() {
					k := pair.Key()
					v := pair.Value()
					var enums []string
					if v.Enum != nil {
						enums = v.Enum
					}
					srv.Variables[k] = ServerVariable{
						Default:     v.Default,
						Description: v.Description,
						Enum:        enums,
					}
				}
			}
			spec.Servers = append(spec.Servers, srv)
		}
	}

	// 3. Tags
	if len(v3Model.Model.Tags) > 0 {
		for _, t := range v3Model.Model.Tags {
			spec.Tags = append(spec.Tags, Tag{
				Name:        t.Name,
				Description: t.Description,
			})
		}
	}

	// 4. Components (Schemas & Security Schemes)
	visitedSchemas := make(map[string]*Schema)
	inProgress := make(map[string]bool)

	if v3Model.Model.Components != nil {
		comps := v3Model.Model.Components

		// Schemas
		if comps.Schemas != nil {
			for pair := comps.Schemas.First(); pair != nil; pair = pair.Next() {
				name := pair.Key()
				proxy := pair.Value()
				refKey := "#/components/schemas/" + name
				schema := convertSchemaProxy(proxy, visitedSchemas, inProgress, 0)
				if schema != nil {
					schema.Ref = refKey
					visitedSchemas[refKey] = schema
					visitedSchemas[name] = schema
					spec.Components.Schemas[name] = schema
				}
			}
		}

		// Security Schemes
		if comps.SecuritySchemes != nil {
			for pair := comps.SecuritySchemes.First(); pair != nil; pair = pair.Next() {
				name := pair.Key()
				ss := pair.Value()
				secScheme := SecurityScheme{
					Type:             ss.Type,
					Description:      ss.Description,
					Name:             ss.Name,
					In:               ss.In,
					Scheme:           ss.Scheme,
					BearerFormat:     ss.BearerFormat,
					OpenIDConnectURL: ss.OpenIdConnectUrl,
				}
				spec.Components.SecuritySchemes[name] = secScheme
			}
		}
	}

	// 5. Paths & Operations
	if v3Model.Model.Paths != nil && v3Model.Model.Paths.PathItems != nil {
		for pair := v3Model.Model.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
			pathKey := pair.Key()
			pathItem := pair.Value()

			ops := extractOperations(pathItem)
			for method, op := range ops {
				if op == nil {
					continue
				}
				endpoint := convertOperation(pathKey, method, op, pathItem, visitedSchemas, inProgress)
				spec.Endpoints = append(spec.Endpoints, endpoint)
			}
		}
	}

	return spec, nil
}

func (p *Parser) loadBytes(pathOrURL string) ([]byte, error) {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, pathOrURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("remote URL returned HTTP %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}

	cleanPath := pathOrURL
	if strings.HasPrefix(cleanPath, "file://") {
		u, err := url.Parse(cleanPath)
		if err == nil {
			cleanPath = u.Path
			cleanPath = strings.TrimPrefix(cleanPath, "/")
		}
	}

	return os.ReadFile(cleanPath)
}

func extractOperations(item *v3.PathItem) map[string]*v3.Operation {
	ops := make(map[string]*v3.Operation)
	if item.Get != nil {
		ops["GET"] = item.Get
	}
	if item.Post != nil {
		ops["POST"] = item.Post
	}
	if item.Put != nil {
		ops["PUT"] = item.Put
	}
	if item.Delete != nil {
		ops["DELETE"] = item.Delete
	}
	if item.Patch != nil {
		ops["PATCH"] = item.Patch
	}
	if item.Head != nil {
		ops["HEAD"] = item.Head
	}
	if item.Options != nil {
		ops["OPTIONS"] = item.Options
	}
	if item.Trace != nil {
		ops["TRACE"] = item.Trace
	}
	return ops
}

func convertOperation(
	pathKey, method string,
	op *v3.Operation,
	pathItem *v3.PathItem,
	visitedSchemas map[string]*Schema,
	inProgress map[string]bool,
) Endpoint {
	epID := op.OperationId
	if epID == "" {
		epID = strings.ToLower(method) + "-" + strings.ReplaceAll(strings.Trim(pathKey, "/"), "/", "-")
	}

	ep := Endpoint{
		ID:          epID,
		Path:        pathKey,
		Method:      method,
		Summary:     op.Summary,
		Description: op.Description,
		OperationID: op.OperationId,
		Tags:        op.Tags,
		Deprecated:  op.Deprecated != nil && *op.Deprecated,
		Responses:   make(map[string]Response),
		Parameters:  make([]Parameter, 0),
		Security:    make([]map[string][]string, 0),
	}

	// Security
	if op.Security != nil {
		for _, secReq := range op.Security {
			if secReq.Requirements != nil {
				m := make(map[string][]string)
				for pair := secReq.Requirements.First(); pair != nil; pair = pair.Next() {
					m[pair.Key()] = pair.Value()
				}
				ep.Security = append(ep.Security, m)
			}
		}
	}

	// Parameters
	paramMap := make(map[string]Parameter)
	for _, p := range pathItem.Parameters {
		if p != nil {
			param := convertParameter(p, visitedSchemas, inProgress)
			paramMap[param.In+":"+param.Name] = param
		}
	}
	for _, p := range op.Parameters {
		if p != nil {
			param := convertParameter(p, visitedSchemas, inProgress)
			paramMap[param.In+":"+param.Name] = param
		}
	}
	for _, param := range paramMap {
		ep.Parameters = append(ep.Parameters, param)
	}

	// Request Body
	if op.RequestBody != nil {
		rb := op.RequestBody
		ep.RequestBody = &RequestBody{
			Description: rb.Description,
			Required:    rb.Required != nil && *rb.Required,
			Content:     make(map[string]MediaType),
		}
		if rb.Content != nil {
			for pair := rb.Content.First(); pair != nil; pair = pair.Next() {
				mediaTypeName := pair.Key()
				mt := pair.Value()
				ep.RequestBody.Content[mediaTypeName] = convertMediaType(mt, visitedSchemas, inProgress)
			}
		}
	}

	// Responses
	if op.Responses != nil && op.Responses.Codes != nil {
		for pair := op.Responses.Codes.First(); pair != nil; pair = pair.Next() {
			code := pair.Key()
			resp := pair.Value()
			respModel := Response{
				StatusCode:  code,
				Description: resp.Description,
				Headers:     make(map[string]Parameter),
				Content:     make(map[string]MediaType),
			}
			if resp.Content != nil {
				for cPair := resp.Content.First(); cPair != nil; cPair = cPair.Next() {
					mtName := cPair.Key()
					mt := cPair.Value()
					respModel.Content[mtName] = convertMediaType(mt, visitedSchemas, inProgress)
				}
			}
			ep.Responses[code] = respModel
		}
	}
	if op.Responses != nil && op.Responses.Default != nil {
		resp := op.Responses.Default
		respModel := Response{
			StatusCode:  "default",
			Description: resp.Description,
			Headers:     make(map[string]Parameter),
			Content:     make(map[string]MediaType),
		}
		if resp.Content != nil {
			for cPair := resp.Content.First(); cPair != nil; cPair = cPair.Next() {
				mtName := cPair.Key()
				mt := cPair.Value()
				respModel.Content[mtName] = convertMediaType(mt, visitedSchemas, inProgress)
			}
		}
		ep.Responses["default"] = respModel
	}

	return ep
}

func convertParameter(p *v3.Parameter, visited map[string]*Schema, inProgress map[string]bool) Parameter {
	req := p.Required != nil && *p.Required
	dep := p.Deprecated
	param := Parameter{
		Name:        p.Name,
		In:          p.In,
		Description: p.Description,
		Required:    req,
		Deprecated:  dep,
		Example:     decodeValue(p.Example),
	}
	if p.Schema != nil {
		param.Schema = convertSchemaProxy(p.Schema, visited, inProgress, 0)
	}
	return param
}

func convertMediaType(mt *v3.MediaType, visited map[string]*Schema, inProgress map[string]bool) MediaType {
	res := MediaType{
		Example:  decodeValue(mt.Example),
		Examples: make(map[string]Example),
	}
	if mt.Schema != nil {
		res.Schema = convertSchemaProxy(mt.Schema, visited, inProgress, 0)
	}
	if mt.Examples != nil {
		for pair := mt.Examples.First(); pair != nil; pair = pair.Next() {
			k := pair.Key()
			ex := pair.Value()
			res.Examples[k] = Example{
				Summary:     ex.Summary,
				Description: ex.Description,
				Value:       decodeValue(ex.Value),
			}
		}
	}
	return res
}

func convertSchemaProxy(
	proxy *base.SchemaProxy,
	visited map[string]*Schema,
	inProgress map[string]bool,
	depth int,
) *Schema {
	if proxy == nil {
		return nil
	}
	if depth > 12 {
		return &Schema{Description: "Recursive reference depth limit reached"}
	}

	ref := proxy.GetReference()
	if ref != "" {
		if inProgress[ref] {
			// Circular reference detected; break cycle
			return &Schema{Ref: ref}
		}
		if cached, ok := visited[ref]; ok {
			return cached
		}
	}

	schema := proxy.Schema()
	if schema == nil {
		if ref != "" {
			return &Schema{Ref: ref}
		}
		return nil
	}

	if ref != "" {
		inProgress[ref] = true
	}

	s := convertBaseSchema(schema, visited, inProgress, depth+1)

	if ref != "" {
		s.Ref = ref
		visited[ref] = s
		delete(inProgress, ref)
	}

	return s
}

func convertBaseSchema(
	schema *base.Schema,
	visited map[string]*Schema,
	inProgress map[string]bool,
	depth int,
) *Schema {
	if schema == nil {
		return nil
	}

	s := &Schema{
		Title:       schema.Title,
		Description: schema.Description,
		Format:      schema.Format,
		Pattern:     schema.Pattern,
		Nullable:    schema.Nullable != nil && *schema.Nullable,
		ReadOnly:    schema.ReadOnly != nil && *schema.ReadOnly,
		WriteOnly:   schema.WriteOnly != nil && *schema.WriteOnly,
		Minimum:     schema.Minimum,
		Maximum:     schema.Maximum,
		MinLength:   schema.MinLength,
		MaxLength:   schema.MaxLength,
		MinItems:    schema.MinItems,
		MaxItems:    schema.MaxItems,
		UniqueItems: schema.UniqueItems != nil && *schema.UniqueItems,
		Required:    schema.Required,
		Properties:  make(map[string]*Schema),
	}

	if len(schema.Type) > 0 {
		s.Type = schema.Type[0]
		if len(schema.Type) > 1 {
			s.Types = schema.Type
		}
	}

	if schema.Default != nil {
		s.Default = decodeValue(schema.Default)
	}
	if schema.Example != nil {
		s.Example = decodeValue(schema.Example)
	}

	if len(schema.Enum) > 0 {
		enums := make([]any, len(schema.Enum))
		for i, v := range schema.Enum {
			enums[i] = decodeValue(v)
		}
		s.Enum = enums
	}

	// Properties
	if schema.Properties != nil {
		for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
			propName := pair.Key()
			propProxy := pair.Value()
			propSchema := convertSchemaProxy(propProxy, visited, inProgress, depth)
			if propSchema != nil {
				s.Properties[propName] = propSchema
			}
		}
	}

	// Items (array)
	if schema.Items != nil && schema.Items.A != nil {
		s.Items = convertSchemaProxy(schema.Items.A, visited, inProgress, depth)
	}

	// Polymorphism (oneOf, anyOf, allOf)
	if len(schema.OneOf) > 0 {
		s.OneOf = make([]*Schema, 0, len(schema.OneOf))
		for _, o := range schema.OneOf {
			if converted := convertSchemaProxy(o, visited, inProgress, depth); converted != nil {
				s.OneOf = append(s.OneOf, converted)
			}
		}
	}
	if len(schema.AnyOf) > 0 {
		s.AnyOf = make([]*Schema, 0, len(schema.AnyOf))
		for _, a := range schema.AnyOf {
			if converted := convertSchemaProxy(a, visited, inProgress, depth); converted != nil {
				s.AnyOf = append(s.AnyOf, converted)
			}
		}
	}
	if len(schema.AllOf) > 0 {
		s.AllOf = make([]*Schema, 0, len(schema.AllOf))
		for _, a := range schema.AllOf {
			if converted := convertSchemaProxy(a, visited, inProgress, depth); converted != nil {
				s.AllOf = append(s.AllOf, converted)
			}
		}
	}

	return s
}

func decodeValue(val any) any {
	if val == nil {
		return nil
	}

	if node, ok := val.(*libyaml.Node); ok {
		if node == nil {
			return nil
		}
		var decoded any
		if err := node.Decode(&decoded); err == nil {
			return normalizeDecoded(decoded)
		}
	}

	return val
}

func normalizeDecoded(v any) any {
	switch val := v.(type) {
	case map[string]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[k] = normalizeDecoded(item)
		}
		return res
	case map[any]any:
		res := make(map[string]any, len(val))
		for k, item := range val {
			res[fmt.Sprintf("%v", k)] = normalizeDecoded(item)
		}
		return res
	case []any:
		res := make([]any, len(val))
		for i, item := range val {
			res[i] = normalizeDecoded(item)
		}
		return res
	default:
		return v
	}
}
