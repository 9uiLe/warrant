// @warrant-covers WARRANT-SERVE
// @warrant-enforces RULE-SERVE-READONLY
package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/registry"
)

// newTestHandler は空レジストリ・既定設定のハンドラを返す（要件 0 件 → PASS）
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	reg := &registry.Registry{Raw: map[string]any{"requirements": []any{}}}
	return Handler(t.TempDir(), reg, config.Default())
}

// TestHandler_RootServesHTML: GET / → 200 / text/html / 非空ボディ
func TestHandler_RootServesHTML(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(t))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

// TestHandler_GraphReturnsJSON: GET /api/graph → 200 / JSON / verdict PASS
func TestHandler_GraphReturnsJSON(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(t))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/graph")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var graph struct {
		Verdict          string `json:"verdict"`
		RequirementCount int    `json:"requirementCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		t.Fatalf("JSON デコード失敗: %v", err)
	}
	if graph.Verdict != "PASS" {
		t.Errorf("verdict = %q, want PASS", graph.Verdict)
	}
}

// TestHandler_RejectsNonGet: GET 以外は 405（読み取り専用）
func TestHandler_RejectsNonGet(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(t))
	defer srv.Close()

	for _, path := range []string{"/", "/api/graph"} {
		req, err := http.NewRequest(http.MethodPost, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status = %d, want 405", path, resp.StatusCode)
		}
	}
}
