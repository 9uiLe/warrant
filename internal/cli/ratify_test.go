// @warrant-covers WARRANT-RATIFY
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/authority"
	"github.com/9uiLe/warrant/internal/config"
)

const ratifyTestRules = `# top comment to test preservation
rules:
  - id: RULE-TEST-RATIFY
    title: "テスト用ルール"
    status: active
    basis: ".warrant/constitution.md#x"
    enforced_by:
      - kind: deterministic
        ref: "internal/cli/ratify_test.go"
`

// makeRatifyRules は未承認(content_hash 無し)のルールを含む rules.yaml を作る
func makeRatifyRules(t *testing.T) (dir, rulesPath string) {
	t.Helper()
	dir = t.TempDir()
	rulesPath = filepath.Join(dir, "rules.yaml")
	if err := os.WriteFile(rulesPath, []byte(ratifyTestRules), 0644); err != nil {
		t.Fatal(err)
	}
	return
}

// isRatified は authority.Run を回して指定 ID のルールが承認済み(Ratified)かを返す。
// これは warrant check の E-RULE-UNRATIFIED 判定と同じ計算を通す受け入れ確認。
func isRatified(t *testing.T, dir, rulesPath, id string) bool {
	t.Helper()
	res, err := authority.Run(dir, rulesPath, config.Default())
	if err != nil {
		t.Fatalf("authority.Run: %v", err)
	}
	for _, r := range res.Rules {
		if r.ID == id {
			return r.Ratified
		}
	}
	t.Fatalf("rule %q not found", id)
	return false
}

// TestRatify_DryRunDoesNotWrite: 既定 dry-run はファイルを変更せず exit 0
func TestRatify_DryRunDoesNotWrite(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)
	before, _ := os.ReadFile(rulesPath)

	var buf bytes.Buffer
	code := ratifyRun(dir, rulesPath, filepath.Join(dir, "config.yaml"), "", false, false, "", &buf)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	after, _ := os.ReadFile(rulesPath)
	if !bytes.Equal(before, after) {
		t.Errorf("dry-run がファイルを変更した")
	}
	if !strings.Contains(buf.String(), "WILL-UPDATE") {
		t.Errorf("dry-run 出力に WILL-UPDATE が無い:\n%s", buf.String())
	}
	if isRatified(t, dir, rulesPath, "RULE-TEST-RATIFY") {
		t.Errorf("dry-run 後に承認済みになっている")
	}
}

// TestRatify_WriteRule: --rule --write で content_hash が書かれ、承認済みになる
func TestRatify_WriteRule(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)

	var buf bytes.Buffer
	code := ratifyRun(dir, rulesPath, filepath.Join(dir, "config.yaml"), "RULE-TEST-RATIFY", false, true, "", &buf)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, buf.String())
	}
	if !isRatified(t, dir, rulesPath, "RULE-TEST-RATIFY") {
		t.Errorf("--write 後も承認済みになっていない")
	}
	// コメントが保全されていること
	out, _ := os.ReadFile(rulesPath)
	if !strings.Contains(string(out), "# top comment to test preservation") {
		t.Errorf("書き込みでコメントが失われた:\n%s", string(out))
	}
}

// TestRatify_Idempotent: 承認済みに再度 --write しても変更なし
func TestRatify_Idempotent(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)
	cfg := filepath.Join(dir, "config.yaml")

	var buf bytes.Buffer
	if code := ratifyRun(dir, rulesPath, cfg, "RULE-TEST-RATIFY", false, true, "", &buf); code != 0 {
		t.Fatalf("1回目 exit = %d", code)
	}
	first, _ := os.ReadFile(rulesPath)

	buf.Reset()
	if code := ratifyRun(dir, rulesPath, cfg, "RULE-TEST-RATIFY", false, true, "", &buf); code != 0 {
		t.Fatalf("2回目 exit = %d", code)
	}
	second, _ := os.ReadFile(rulesPath)

	if !bytes.Equal(first, second) {
		t.Errorf("冪等でない: 2回目の書き込みでファイルが変化した")
	}
	if !strings.Contains(buf.String(), "更新対象はありません") {
		t.Errorf("2回目に MATCH 扱いになっていない:\n%s", buf.String())
	}
}

// TestRatify_WriteRequiresSelector: --write 単独(対象未指定)は exit 2
func TestRatify_WriteRequiresSelector(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)
	before, _ := os.ReadFile(rulesPath)

	var buf bytes.Buffer
	code := ratifyRun(dir, rulesPath, filepath.Join(dir, "config.yaml"), "", false, true, "", &buf)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	after, _ := os.ReadFile(rulesPath)
	if !bytes.Equal(before, after) {
		t.Errorf("拒否されたのにファイルが変更された")
	}
}

// TestRatify_UnknownRule: 存在しない --rule は exit 2
func TestRatify_UnknownRule(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)
	var buf bytes.Buffer
	code := ratifyRun(dir, rulesPath, filepath.Join(dir, "config.yaml"), "RULE-DOES-NOT-EXIST", false, true, "", &buf)
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

// TestRatify_All: --all --write で全ルールが承認される
func TestRatify_All(t *testing.T) {
	dir, rulesPath := makeRatifyRules(t)
	var buf bytes.Buffer
	code := ratifyRun(dir, rulesPath, filepath.Join(dir, "config.yaml"), "", true, true, "approver@example.com", &buf)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, buf.String())
	}
	if !isRatified(t, dir, rulesPath, "RULE-TEST-RATIFY") {
		t.Errorf("--all --write 後も承認済みになっていない")
	}
	out, _ := os.ReadFile(rulesPath)
	if !strings.Contains(string(out), "approver@example.com") {
		t.Errorf("approved_by が記録されていない:\n%s", string(out))
	}
}
