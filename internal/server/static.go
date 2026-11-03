package server

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist/*
var staticFS embed.FS

func (s *APIServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	distFS, err := fs.Sub(staticFS, "dist")
	if err != nil {
		http.Error(w, "Failed to load static assets", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Try serving existing asset (e.g. assets/index.js, assets/style.css, favicon.ico)
	f, err := distFS.Open(path)
	if err == nil {
		_ = f.Close()
		http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
		return
	}

	// Fallback to index.html for Single Page Application client-side routing
	indexFile, err := distFS.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer indexFile.Close()

	stat, err := indexFile.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	seeker, ok := indexFile.(io.ReadSeeker)
	if !ok {
		// Read fully if not a seeker
		data, readErr := io.ReadAll(indexFile)
		if readErr != nil {
			http.Error(w, "Failed to read index.html", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
		return
	}

	http.ServeContent(w, r, "index.html", stat.ModTime(), seeker)
}
