package codegen

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateGoNetHTTP generates idiomatic Go code using standard library `net/http`.
func GenerateGoNetHTTP(method, targetURL string, headers map[string]string, body string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "GET"
	}

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"io\"\n")
	sb.WriteString("\t\"net/http\"\n")
	if body != "" && method != "GET" && method != "HEAD" {
		sb.WriteString("\t\"strings\"\n")
	}
	sb.WriteString(")\n\n")
	sb.WriteString("func main() {\n")
	sb.WriteString(fmt.Sprintf("\turl := %q\n", targetURL))

	if body != "" && method != "GET" && method != "HEAD" {
		sb.WriteString(fmt.Sprintf("\tbody := strings.NewReader(%q)\n", body))
		sb.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(%q, url, body)\n", method))
	} else {
		sb.WriteString(fmt.Sprintf("\treq, err := http.NewRequest(%q, url, nil)\n", method))
	}
	sb.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n\n")

	if len(headers) > 0 {
		var keys []string
		for k := range headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("\treq.Header.Set(%q, %q)\n", k, headers[k]))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\tclient := &http.Client{}\n")
	sb.WriteString("\tresp, err := client.Do(req)\n")
	sb.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n")
	sb.WriteString("\tdefer resp.Body.Close()\n\n")
	sb.WriteString("\trespBody, err := io.ReadAll(resp.Body)\n")
	sb.WriteString("\tif err != nil {\n\t\tpanic(err)\n\t}\n\n")
	sb.WriteString("\tfmt.Printf(\"Status: %s\\n\", resp.Status)\n")
	sb.WriteString("\tfmt.Println(string(respBody))\n")
	sb.WriteString("}\n")

	return sb.String()
}
