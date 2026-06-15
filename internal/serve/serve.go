package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/registry"
	"github.com/9uiLe/warrant/internal/web"
)

// Start は 127.0.0.1 に限定した読み取り専用 HTTP サーバを起動する
func Start(addr, root string, reg *registry.Registry, cfg *config.Config) error {
	mux := http.NewServeMux()

	// GET / → index.html (go:embed)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(web.IndexHTML)
	})

	// GET /api/graph → その都度 SSOT から再計算
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		result, err := check.Run(root, reg, cfg)
		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		generatedAt := time.Now().UTC().Format(time.RFC3339)
		graph := check.BuildGraph(result.Requirements, result.Violations, generatedAt)

		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(graph); err != nil {
			http.Error(w, "json error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(buf.Bytes())
	})

	return http.ListenAndServe(addr, mux)
}
