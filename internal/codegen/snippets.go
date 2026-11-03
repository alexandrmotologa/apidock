package codegen

// SnippetSet encapsulates ready-to-copy code snippets across popular languages.
type SnippetSet struct {
	Curl            string `json:"curl"`
	TypeScriptFetch string `json:"typescriptFetch"`
	TypeScriptAxios string `json:"typescriptAxios"`
	PythonRequests  string `json:"pythonRequests"`
	PythonHTTPX     string `json:"pythonHttpx"`
	GoNetHTTP       string `json:"goNetHttp"`
}

// GenerateAll produces snippets for all supported environments.
func GenerateAll(method, targetURL string, headers map[string]string, body string) SnippetSet {
	return SnippetSet{
		Curl:            GenerateCurl(method, targetURL, headers, body),
		TypeScriptFetch: GenerateTypeScriptFetch(method, targetURL, headers, body),
		TypeScriptAxios: GenerateTypeScriptAxios(method, targetURL, headers, body),
		PythonRequests:  GeneratePythonRequests(method, targetURL, headers, body),
		PythonHTTPX:     GeneratePythonHTTPX(method, targetURL, headers, body),
		GoNetHTTP:       GenerateGoNetHTTP(method, targetURL, headers, body),
	}
}
