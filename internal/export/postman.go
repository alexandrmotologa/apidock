package export

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

// PostmanCollection represents the top-level Postman 2.1 collection format.
type PostmanCollection struct {
	Info     PostmanInfo       `json:"info"`
	Item     []PostmanItem     `json:"item"`
	Variable []PostmanVariable `json:"variable,omitempty"`
}

type PostmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

type PostmanItem struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Item        []PostmanItem    `json:"item,omitempty"`
	Request     *PostmanRequest  `json:"request,omitempty"`
	Response    []map[string]any `json:"response,omitempty"`
}

type PostmanRequest struct {
	Method      string               `json:"method"`
	Header      []PostmanHeader      `json:"header"`
	Body        *PostmanRequestBody  `json:"body,omitempty"`
	URL         PostmanURL           `json:"url"`
	Description string               `json:"description,omitempty"`
}

type PostmanHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type PostmanRequestBody struct {
	Mode    string            `json:"mode"`
	Raw     string            `json:"raw,omitempty"`
	Options map[string]any    `json:"options,omitempty"`
}

type PostmanURL struct {
	Raw   string             `json:"raw"`
	Host  []string           `json:"host,omitempty"`
	Path  []string           `json:"path,omitempty"`
	Query []PostmanQueryItem `json:"query,omitempty"`
}

type PostmanQueryItem struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type PostmanVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ToPostmanCollection converts an OpenAPI Spec into a Postman Collection v2.1.
func ToPostmanCollection(s *spec.Spec) (*PostmanCollection, error) {
	col := &PostmanCollection{
		Info: PostmanInfo{
			PostmanID:   randomUUID(),
			Name:        s.Info.Title,
			Description: s.Info.Description,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Item:     make([]PostmanItem, 0),
		Variable: make([]PostmanVariable, 0),
	}

	baseUrl := "http://localhost:8080"
	if len(s.Servers) > 0 {
		baseUrl = s.Servers[0].URL
	}
	col.Variable = append(col.Variable, PostmanVariable{
		Key:   "baseUrl",
		Value: baseUrl,
		Type:  "string",
	})

	// Group endpoints by primary tag or "Default"
	tagMap := make(map[string][]spec.Endpoint)
	for _, ep := range s.Endpoints {
		tag := "General"
		if len(ep.Tags) > 0 {
			tag = ep.Tags[0]
		}
		tagMap[tag] = append(tagMap[tag], ep)
	}

	for tag, endpoints := range tagMap {
		folder := PostmanItem{
			Name: tag,
			Item: make([]PostmanItem, 0),
		}

		for _, ep := range endpoints {
			reqItem := convertEndpointToPostmanItem(ep)
			folder.Item = append(folder.Item, reqItem)
		}

		col.Item = append(col.Item, folder)
	}

	return col, nil
}

func convertEndpointToPostmanItem(ep spec.Endpoint) PostmanItem {
	name := ep.Summary
	if name == "" {
		name = fmt.Sprintf("%s %s", strings.ToUpper(ep.Method), ep.Path)
	}

	// Prepare path and variables
	cleanPath := strings.TrimPrefix(ep.Path, "/")
	pathSegments := strings.Split(cleanPath, "/")

	rawUrl := "{{baseUrl}}" + ep.Path
	var queryItems []PostmanQueryItem
	var headers []PostmanHeader

	for _, param := range ep.Parameters {
		switch strings.ToLower(param.In) {
		case "query":
			exampleVal := ""
			if param.Example != nil {
				exampleVal = fmt.Sprintf("%v", param.Example)
			}
			queryItems = append(queryItems, PostmanQueryItem{
				Key:         param.Name,
				Value:       exampleVal,
				Description: param.Description,
			})
		case "header":
			headers = append(headers, PostmanHeader{
				Key:         param.Name,
				Value:       "",
				Description: param.Description,
			})
		}
	}

	headers = append(headers, PostmanHeader{
		Key:   "Accept",
		Value: "application/json",
	})

	var reqBody *PostmanRequestBody
	if ep.RequestBody != nil && len(ep.RequestBody.Content) > 0 {
		for mime, mt := range ep.RequestBody.Content {
			if strings.Contains(mime, "json") {
				headers = append(headers, PostmanHeader{
					Key:   "Content-Type",
					Value: mime,
				})
				bodyRaw := "{}"
				if mt.Example != nil {
					if b, err := json.MarshalIndent(mt.Example, "", "  "); err == nil {
						bodyRaw = string(b)
					}
				}
				reqBody = &PostmanRequestBody{
					Mode: "raw",
					Raw:  bodyRaw,
					Options: map[string]any{
						"raw": map[string]string{
							"language": "json",
						},
					},
				}
				break
			}
		}
	}

	return PostmanItem{
		Name:        name,
		Description: ep.Description,
		Request: &PostmanRequest{
			Method:      strings.ToUpper(ep.Method),
			Header:      headers,
			Body:        reqBody,
			Description: ep.Description,
			URL: PostmanURL{
				Raw:   rawUrl,
				Host:  []string{"{{baseUrl}}"},
				Path:  pathSegments,
				Query: queryItems,
			},
		},
		Response: make([]map[string]any, 0),
	}
}

// ToPostmanJSON serializes a spec to Postman Collection JSON format.
func ToPostmanJSON(s *spec.Spec) ([]byte, error) {
	col, err := ToPostmanCollection(s)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(col, "", "  ")
}
