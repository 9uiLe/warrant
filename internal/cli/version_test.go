package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/9uiLe/warrant/internal/version"
)

// TestRunVersion_ExitCode: runVersion は exit 0 を返す
func TestRunVersion_ExitCode(t *testing.T) {
	if code := runVersion(nil); code != 0 {
		t.Fatalf("runVersion exit = %d, want 0", code)
	}
}

// TestRunVersion_OutputContainsWarrant: 出力に "warrant" が含まれる
func TestRunVersion_OutputContainsWarrant(t *testing.T) {
	// 標準出力をキャプチャする
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runVersion(nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "warrant") {
		t.Errorf("出力に 'warrant' が含まれない: got %q", out)
	}
}

// TestRunVersion_ContainsVersion: 出力に version.Version の値が含まれる
func TestRunVersion_ContainsVersion(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runVersion(nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	want := fmt.Sprintf("warrant %s", version.Version)
	if !strings.Contains(out, want) {
		t.Errorf("出力に %q が含まれない: got %q", want, out)
	}
}
