package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesChatThroughSameOriginProxy(t *testing.T) {
	var seenPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer upstream.Close()

	server := httptest.NewServer(Handler(upstream.URL))
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/chat", "application/json", strings.NewReader(`{"model":"llama3"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if seenPath != "/api/chat" {
		t.Fatalf("expected /api/chat proxy path, got %q", seenPath)
	}
	if string(body) != `{"message":"ok"}` {
		t.Fatalf("unexpected body %s", string(body))
	}
}
