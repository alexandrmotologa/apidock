package export

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

var (
	cssLinkRegex   = regexp.MustCompile(`<link[^>]*rel="stylesheet"[^>]*href="(/assets/[^"]+)"[^>]*>`)
	scriptSrcRegex = regexp.MustCompile(`<script[^>]*src="(/assets/[^"]+)"[^>]*></script>`)
)

// BuildSingleFileHTML compiles the React application and OpenAPI spec into a zero-dependency standalone HTML document.
func BuildSingleFileHTML(s *spec.Spec, distFS fs.FS) ([]byte, error) {
	indexFile, err := distFS.Open("index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to open index.html from dist: %w", err)
	}
	defer indexFile.Close()

	rawHTMLBytes, err := io.ReadAll(indexFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.html: %w", err)
	}
	html := string(rawHTMLBytes)

	// 1. Inline CSS stylesheets
	cssMatches := cssLinkRegex.FindAllStringSubmatch(html, -1)
	for _, match := range cssMatches {
		fullTag := match[0]
		assetPath := strings.TrimPrefix(match[1], "/")

		cssBytes, err := readAssetFile(distFS, assetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CSS asset %s: %w", assetPath, err)
		}

		styleTag := fmt.Sprintf("<style>\n%s\n</style>", string(cssBytes))
		html = strings.Replace(html, fullTag, styleTag, 1)
	}

	// 2. Inline Javascript bundles
	jsMatches := scriptSrcRegex.FindAllStringSubmatch(html, -1)
	for _, match := range jsMatches {
		fullTag := match[0]
		assetPath := strings.TrimPrefix(match[1], "/")

		jsBytes, err := readAssetFile(distFS, assetPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read JS asset %s: %w", assetPath, err)
		}

		scriptTag := fmt.Sprintf("<script type=\"module\">\n%s\n</script>", string(jsBytes))
		html = strings.Replace(html, fullTag, scriptTag, 1)
	}

	// 3. Serialize OpenAPI Spec JSON
	specJSON, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize spec payload: %w", err)
	}

	embeddedSpecScript := fmt.Sprintf(
		"<script id=\"apidock-spec\" type=\"application/json\">\n%s\n</script>\n",
		string(specJSON),
	)

	// Inject spec payload right before the root mount element
	if strings.Contains(html, "<div id=\"root\">") {
		html = strings.Replace(html, "<div id=\"root\">", embeddedSpecScript+"<div id=\"root\">", 1)
	} else {
		html = strings.Replace(html, "</body>", embeddedSpecScript+"</body>", 1)
	}

	// Update document title if spec info title exists
	if s.Info.Title != "" {
		newTitle := fmt.Sprintf("<title>%s - APIDock</title>", s.Info.Title)
		titleRegex := regexp.MustCompile(`<title>.*?</title>`)
		if titleRegex.MatchString(html) {
			html = titleRegex.ReplaceAllString(html, newTitle)
		}
	}

	return []byte(html), nil
}

func readAssetFile(distFS fs.FS, path string) ([]byte, error) {
	f, err := distFS.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
