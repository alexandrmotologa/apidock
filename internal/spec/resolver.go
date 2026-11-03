package spec

import (
	"path/filepath"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
)

// Resolver handles document configuration and multi-file reference settings.
type Resolver struct {
	basePath string
}

// NewResolver initializes a Resolver with an optional root path.
func NewResolver(basePath string) *Resolver {
	return &Resolver{
		basePath: basePath,
	}
}

// BuildDocument creates a libopenapi Document with proper base path configuration
// to resolve relative multi-file $ref pointers.
func (r *Resolver) BuildDocument(data []byte, sourcePath string) (libopenapi.Document, error) {
	config := &datamodel.DocumentConfiguration{}

	if sourcePath != "" && !strings.HasPrefix(sourcePath, "http://") && !strings.HasPrefix(sourcePath, "https://") {
		dir := filepath.Dir(sourcePath)
		absDir, err := filepath.Abs(dir)
		if err == nil {
			config.BasePath = absDir
		}
	}

	return libopenapi.NewDocumentWithConfiguration(data, config)
}
