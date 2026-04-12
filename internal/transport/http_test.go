package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPTransportMCPOptionsPreflight(t *testing.T) {
	transport := &HTTPTransport{}
	mux := transport.newMux(context.Background(), ServerInfo{Name: "test", Version: "1.1.0"}, func(ctx context.Context, request json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`{"ok":true}`), nil
	})

	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if got := res.Header().Get("Access-Control-Allow-Methods"); got != "POST, OPTIONS" {
		t.Fatalf("unexpected allow methods header: %q", got)
	}
}
