// @warrant-covers WARRANT-INIT
package cli

import (
	"os"
	"path/filepath"
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
