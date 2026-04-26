package ui

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed static/index.html
var indexHTML string

func Handler(proxyBase string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		h := strings.ReplaceAll(indexHTML, "__PROXY_BASE__", proxyBase)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(h))
	})
	return mux
}
