package spec

// Spec represents the normalized, fully-resolved OpenAPI 3.0/3.1 document
// ready for consumption by both the web studio and the mock server engine.
type Spec struct {
	OpenAPI    string                     `json:"openapi"`
	Info       Info                       `json:"info"`
	Servers    []Server                   `json:"servers"`
	Tags       []Tag                      `json:"tags"`
	Endpoints  []Endpoint                 `json:"endpoints"`
	Components Components                 `json:"components"`
	Errors     []string                   `json:"errors,omitempty"`
	Warnings   []string                   `json:"warnings,omitempty"`
}

type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Version        string   `json:"version"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

type ServerVariable struct {
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Endpoint struct {
	ID          string                       `json:"id"`
	Path        string                       `json:"path"`
	Method      string                       `json:"method"`
	Summary     string                       `json:"summary,omitempty"`
	Description string                       `json:"description,omitempty"`
	OperationID string                       `json:"operationId,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Deprecated  bool                         `json:"deprecated,omitempty"`
	Security    []map[string][]string        `json:"security,omitempty"`
	Parameters  []Parameter                  `json:"parameters,omitempty"`
	RequestBody *RequestBody                 `json:"requestBody,omitempty"`
	Responses   map[string]Response          `json:"responses,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "query", "header", "path", "cookie"
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required"`
	Deprecated  bool    `json:"deprecated,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     any     `json:"example,omitempty"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required"`
	Content     map[string]MediaType `json:"content"`
}

type Response struct {
	StatusCode  string               `json:"statusCode"`
	Description string               `json:"description"`
	Headers     map[string]Parameter `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema   *Schema            `json:"schema,omitempty"`
	Example  any                `json:"example,omitempty"`
	Examples map[string]Example `json:"examples,omitempty"`
}

type Example struct {
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	Value       any    `json:"value,omitempty"`
}

type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Types                []string           `json:"types,omitempty"` // For OpenAPI 3.1 multi-types
	Format               string             `json:"format,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              any                `json:"default,omitempty"`
	Example              any                `json:"example,omitempty"`
	Examples             []any              `json:"examples,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	ExclusiveMinimum     any                `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     any                `json:"exclusiveMaximum,omitempty"`
	MinLength            *int64             `json:"minLength,omitempty"`
	MaxLength            *int64             `json:"maxLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MinItems             *int64             `json:"minItems,omitempty"`
	MaxItems             *int64             `json:"maxItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema       `json:"schemas,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
}

type SecurityScheme struct {
	Type             string            `json:"type"` // "apiKey", "http", "oauth2", "openIdConnect"
	Description      string            `json:"description,omitempty"`
	Name             string            `json:"name,omitempty"` // for apiKey
	In               string            `json:"in,omitempty"`   // "query", "header", "cookie"
	Scheme           string            `json:"scheme,omitempty"` // "bearer", "basic", etc.
	BearerFormat     string            `json:"bearerFormat,omitempty"`
	Flows            map[string]OAuthFlow `json:"flows,omitempty"`
	OpenIDConnectURL string            `json:"openIdConnectUrl,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}
