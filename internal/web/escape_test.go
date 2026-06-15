package web

import (
	"bytes"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/projection"
)

// TestTemplate_EscapesUserData は、要件タイトル・カテゴリ・ファイルパス等の
// SSOT 由来データがそのまま HTML に注入されない（html/template のオートエスケープが
// 効いている）ことを保証する回帰テスト。text/template や template.HTML への退行を検知する。
func TestTemplate_EscapesUserData(t *testing.T) {
	rep := projection.Report{
		Meta: projection.ReportMeta{
			GeneratedAt: "2026-06-15T00:00:00Z",
			Repository:  "<b>repo</b>",
			Branch:      "feat/<svg>",
			Commit:      "deadbeef",
		},
		Summary: projection.ReportSummary{Total: 1, Passed: 1, CoveragePercent: 100},
		Requirements: []projection.ReportRequirement{
			{
				ID:       "XSS-001",
				Title:    "<script>alert(1)</script>",
				Status:   "active",
				Priority: "critical",
				Category: "</span><img src=x onerror=alert(2)>",
				Verdict:  "pass",
				Tests:    []projection.ReportTest{{File: "x_test.go\"><script>1</script>", Status: "linked"}},
				Spec:     projection.ReportSpec{Doc: "docs/<x>.md", Section: "## <b>", Status: "linked"},
			},
		},
	}

	var buf bytes.Buffer
	if err := Template.Execute(&buf, rep); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	out := buf.String()

	// 生のスクリプト・属性インジェクションが残っていてはならない。
	forbidden := []string{
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(2)>",
		"<script>1</script>",
	}
	for _, f := range forbidden {
		if strings.Contains(out, f) {
			t.Errorf("生の危険文字列が出力に含まれている（オートエスケープ未適用）: %q", f)
		}
	}

	// エスケープ済みの痕跡が存在すること（描画自体は行われている）。
	if !strings.Contains(out, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("エスケープ済みタイトルが出力に見つからない")
	}
}
