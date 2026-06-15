// @warrant-covers WARRANT-INIT
package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunInit_CreatesFiles: 空ディレクトリで init すると .warrant/ に 4 ファイルが作られ exit 0
func TestRunInit_CreatesFiles(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir}); code != 0 {
		t.Fatalf("runInit exit = %d, want 0", code)
	}

	want := []string{
		"config.yaml",
		"requirements.yaml",
		"requirements.schema.json",
		"README.md",
	}
	for _, name := range want {
		path := filepath.Join(dir, ".warrant", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf(".warrant/%s が作られていない: %v", name, err)
		}
	}
}

// TestRunInit_Idempotent: 既存ファイルは上書きしない（べき等）
func TestRunInit_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir}); code != 0 {
		t.Fatalf("1 回目 runInit exit = %d, want 0", code)
	}

	// ユーザーが編集した想定で config.yaml を書き換える
	cfgPath := filepath.Join(dir, ".warrant", "config.yaml")
	sentinel := "# user edited\n"
	if err := os.WriteFile(cfgPath, []byte(sentinel), 0644); err != nil {
		t.Fatal(err)
	}

	// 2 回目: 既存ファイルはスキップされ、内容は保持される
	if code := runInit([]string{"--repo-root", dir}); code != 0 {
		t.Fatalf("2 回目 runInit exit = %d, want 0", code)
	}

	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != sentinel {
		t.Errorf("既存 config.yaml が上書きされた: got %q, want %q", string(got), sentinel)
	}
}

// TestRunInit_DefaultLang_Go: --lang 無指定で go プリセットの test_globs になる
func TestRunInit_DefaultLang_Go(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir}); code != 0 {
		t.Fatalf("runInit exit = %d, want 0", code)
	}

	cfgPath := filepath.Join(dir, ".warrant", "config.yaml")
	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)

	// go プリセットの有効 glob が存在すること
	if !strings.Contains(content, `"**/*_test.go"`) {
		t.Errorf("go プリセットの glob が含まれていない: %s", content)
	}
}

// TestRunInit_LangSwift: --lang swift で **/*Tests.swift が有効行に含まれる
func TestRunInit_LangSwift(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir, "--lang", "swift"}); code != 0 {
		t.Fatalf("runInit exit = %d, want 0", code)
	}

	cfgPath := filepath.Join(dir, ".warrant", "config.yaml")
	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)

	// swift の有効 glob が含まれること
	if !strings.Contains(content, `"**/*Tests.swift"`) {
		t.Errorf("swift プリセットの glob が含まれていない: %s", content)
	}

	// go の glob が有効行（`  - "..."` 形式）として含まれないこと
	// コメント行には全言語が含まれるため、有効行のみを確認する
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") && strings.Contains(trimmed, `"**/*_test.go"`) {
			t.Errorf("swift プリセットなのに go の glob が有効行に含まれている: %s", line)
		}
	}
}

// TestRunInit_UnknownLang: 未知の --lang ruby で exit 2 になる
func TestRunInit_UnknownLang(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir, "--lang", "ruby"}); code != 2 {
		t.Fatalf("runInit exit = %d, want 2", code)
	}
}

// TestRunInit_SelfDescriptiveConfig: 生成 config に全言語のコメント例が含まれる
func TestRunInit_SelfDescriptiveConfig(t *testing.T) {
	dir := t.TempDir()

	if code := runInit([]string{"--repo-root", dir}); code != 0 {
		t.Fatalf("runInit exit = %d, want 0", code)
	}

	cfgPath := filepath.Join(dir, ".warrant", "config.yaml")
	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)

	// 全言語のコメント例が含まれること
	for _, keyword := range []string{"swift", "python", "js"} {
		if !strings.Contains(content, keyword) {
			t.Errorf("config.yaml に %s のコメント例が含まれていない", keyword)
		}
	}
}
