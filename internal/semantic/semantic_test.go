// @warrant-covers WARRANT-ADVISE
package semantic_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/9uiLe/warrant/internal/semantic"
)

func writeScript(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "judge.sh")
	if err := os.WriteFile(p, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExecJudge_Pass(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\necho '{\"verdict\":\"pass\",\"rationale\":\"ok\"}'")
	j, err := semantic.NewExecJudge(script, 5)
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.Judge(context.Background(), semantic.Request{RuleID: "R-1"})
	if err != nil {
		t.Fatal(err)
	}
	if v.Verdict != "pass" {
		t.Errorf("verdict: got %q, want %q", v.Verdict, "pass")
	}
}

func TestExecJudge_Fail(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\necho '{\"verdict\":\"fail\",\"rationale\":\"bad\"}'")
	j, err := semantic.NewExecJudge(script, 5)
	if err != nil {
		t.Fatal(err)
	}
	v, err := j.Judge(context.Background(), semantic.Request{RuleID: "R-2"})
	if err != nil {
		t.Fatal(err)
	}
	if v.Verdict != "fail" {
		t.Errorf("verdict: got %q, want %q", v.Verdict, "fail")
	}
}

func TestExecJudge_JSONRoundTrip(t *testing.T) {
	// Script reads stdin and checks it is non-empty, then outputs a fixed verdict
	script := writeScript(t, `#!/bin/sh
input=$(cat)
if [ -z "$input" ]; then
  echo '{"verdict":"fail","rationale":"no stdin"}'
else
  echo '{"verdict":"pass","rationale":"got stdin"}'
fi
`)
	j, err := semantic.NewExecJudge(script, 5)
	if err != nil {
		t.Fatal(err)
	}
	req := semantic.Request{RuleID: "R-3", Title: "test", Criterion: "check"}
	v, err := j.Judge(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if v.Verdict != "pass" {
		t.Errorf("verdict: got %q, want pass", v.Verdict)
	}
	// Rationale should be non-empty
	if v.Rationale == "" {
		t.Error("rationale should not be empty")
	}
}

func TestExecJudge_Timeout(t *testing.T) {
	script := writeScript(t, "#!/bin/sh\nsleep 10")
	j, err := semantic.NewExecJudge(script, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = j.Judge(context.Background(), semantic.Request{RuleID: "R-4"})
	if err == nil {
		t.Fatal("expected error due to timeout, got nil")
	}
}

func TestNewExecJudge_NoCommand(t *testing.T) {
	_, err := semantic.NewExecJudge("", 5)
	if !errors.Is(err, semantic.ErrNoCommand) {
		t.Errorf("expected ErrNoCommand, got %v", err)
	}
}
