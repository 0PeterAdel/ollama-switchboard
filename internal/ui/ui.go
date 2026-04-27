package ui

import (
	_ "embed"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

//go:embed static/index.html
var indexHTML string

func Handler(proxyAddress string) http.Handler {
	mux := http.NewServeMux()
	proxy := proxyHandler(proxyAddress)
	mux.Handle("/api/", proxy)
	mux.Handle("/v1/", proxy)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	return mux
}

func proxyHandler(proxyAddress string) http.Handler {
	target, err := proxyURL(proxyAddress)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "proxy target unavailable", http.StatusBadGateway)
		})
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "proxy upstream unavailable", http.StatusBadGateway)
	}
	return rp
}

func proxyURL(address string) (*url.URL, error) {
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return url.Parse(address)
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return url.Parse("http://" + address)
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return url.Parse("http://" + net.JoinHostPort(host, port))
}
