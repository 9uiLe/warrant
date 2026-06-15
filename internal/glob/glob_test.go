package glob

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// mkTree はテンポラリディレクトリにファイルツリーを作成するヘルパー。
func mkTree(t *testing.T, files []string) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdirall %s: %v", abs, err)
		}
		if err := os.WriteFile(abs, []byte{}, 0o644); err != nil {
			t.Fatalf("writefile %s: %v", abs, err)
		}
	}
	return root
}

func TestGlob(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		pattern string
		want    []string
	}{
		{
			name: "app/**/*Tests*.swift matches nested and direct",
			files: []string{
				"app/Foo/BarTests.swift",
				"app/BarTests.swift",
				"app/Foo/NotATest.swift",
			},
			pattern: "app/**/*Tests*.swift",
			want:    []string{"app/BarTests.swift", "app/Foo/BarTests.swift"},
		},
		{
			name: ".git dir not matched by **/x",
			files: []string{
				".git/x",
				"src/x",
			},
			pattern: "**/x",
			want:    []string{"src/x"},
		},
		{
			name: ".warrant dir not matched by **/*.yaml",
			files: []string{
				".warrant/config.yaml",
				"docs/spec.yaml",
			},
			pattern: "**/*.yaml",
			want:    []string{"docs/spec.yaml"},
		},
		{
			name: "a/b/c.go matched by **/*.go",
			files: []string{
				"a/b/c.go",
				"other.txt",
			},
			pattern: "**/*.go",
			want:    []string{"a/b/c.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := mkTree(t, tt.files)
			got, err := Glob(root, tt.pattern)
			if err != nil {
				t.Fatalf("Glob error: %v", err)
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%q, want[%d]=%q", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}

func TestFnMatch(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		pattern string
		want    bool
	}{
		{
			name:    "simple extension match",
			input:   "hello.txt",
			pattern: "*.txt",
			want:    true,
		},
		{
			name:    "star crosses slash",
			input:   "path/to/file.txt",
			pattern: "*.txt",
			want:    true,
		},
		{
			name:    "case sensitive mismatch",
			input:   "Hello.txt",
			pattern: "*.TXT",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FnMatch(tt.input, tt.pattern)
			if got != tt.want {
				t.Errorf("FnMatch(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.want)
			}
		})
	}
}
