package serve

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/gitmeta"
	"github.com/9uiLe/warrant/internal/registry"
	"github.com/9uiLe/warrant/internal/web"
)

// Start は 127.0.0.1 に限定した読み取り専用 HTTP サーバを起動する
func Start(addr, root string, reg *registry.Registry, cfg *config.Config) error {
	return http.ListenAndServe(addr, Handler(root, reg, cfg))
}

// Handler は serve が公開する読み取り専用ルーティングを構築する。
// GET 以外は 405 を返し、状態を変更する経路を持たない。
func Handler(root string, reg *registry.Registry, cfg *config.Config) http.Handler {
	mux := http.NewServeMux()

	// GET / → SSOT から再計算し html/template でサーバサイド描画
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		result, err := check.Run(root, reg, cfg)
		if err != nil {
			http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		meta := gitmeta.GitMeta(root)
		meta.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

		rep := check.BuildReport(result.Requirements, result.Violations, meta, meta.GeneratedAt)

		var buf bytes.Buffer
		if err := web.Template.Execute(&buf, rep); err != nil {
			http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
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

	return mux
}
