package httpserver

import (
	"embed"
	"net/http"
)

//go:embed web/*
var webAssets embed.FS

func (s *Server) webIndex(w http.ResponseWriter, r *http.Request) {
	serveEmbeddedAsset(w, "web/index.html", "text/html; charset=utf-8", false)
}

func (s *Server) webStyles(w http.ResponseWriter, r *http.Request) {
	serveEmbeddedAsset(w, "web/app.css", "text/css; charset=utf-8", true)
}

func (s *Server) webScript(w http.ResponseWriter, r *http.Request) {
	serveEmbeddedAsset(w, "web/app.js", "application/javascript; charset=utf-8", true)
}

func serveEmbeddedAsset(w http.ResponseWriter, name, contentType string, cache bool) {
	data, err := webAssets.ReadFile(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "web asset could not be loaded")
		return
	}
	w.Header().Set("Content-Type", contentType)
	if cache {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
