// @warrant-covers WARRANT-ADVISE
package cli_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/9uiLe/warrant/internal/cli"
	"github.com/9uiLe/warrant/internal/semantic"
)

type mockJudge struct {
	verdict semantic.Verdict
	err     error
	calls   int
	lastReq semantic.Request
}

func (m *mockJudge) Judge(ctx context.Context, req semantic.Request) (semantic.Verdict, error) {
	m.calls++
	m.lastReq = req
	return m.verdict, m.err
}

// makeMinimalRulesYAML creates a rules.yaml. withSemantic=true で semantic エントリを
// 1 件含めることで、advise の judge 呼び出しパスを実際に走らせる。
// authority.Run は basis 欠落などの違反があってもルール自体は Result.Rules に返すため、
// テスト用に basis ファイル・承認ハッシュを用意しなくても judge は呼ばれる。
func makeMinimalRulesYAML(t *testing.T, withSemantic bool) (dir, rulesPath string) {
	t.Helper()
	dir = t.TempDir()
	rulesPath = filepath.Join(dir, "rules.yaml")

	content := `rules: []`
	if withSemantic {
		content = `rules:
  - id: RULE-TEST-SEMANTIC
    title: "テスト用 semantic ルール"
    basis: ".warrant/constitution.md#x"
    enforced_by:
      - kind: semantic
        ref: ".warrant/assertions/test.md"
        criterion: "テスト基準"
        governs:
          - "*.go"
`
	}
	if err := os.WriteFile(rulesPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return
}

func TestRunAdvise_NoJudge_ExitsZero(t *testing.T) {
	dir, rulesPath := makeMinimalRulesYAML(t, false)
	code := cli.RunAdviseWithJudge(dir, rulesPath, nil)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
}

func TestRunAdvise_JudgePass_ExitsZero(t *testing.T) {
	dir, rulesPath := makeMinimalRulesYAML(t, true)
	j := &mockJudge{verdict: semantic.Verdict{Verdict: "pass", Rationale: "all good"}}
	code := cli.RunAdviseWithJudge(dir, rulesPath, j)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// semantic エントリに対して judge が実際に呼ばれたことを検証(効いている保証)
	if j.calls != 1 {
		t.Errorf("expected judge to be called once, got %d", j.calls)
	}
	if j.lastReq.RuleID != "RULE-TEST-SEMANTIC" || j.lastReq.Criterion != "テスト基準" {
		t.Errorf("judge received unexpected request: %+v", j.lastReq)
	}
}

func TestRunAdvise_JudgeFail_ExitsZero(t *testing.T) {
	dir, rulesPath := makeMinimalRulesYAML(t, true)
	j := &mockJudge{verdict: semantic.Verdict{Verdict: "fail", Rationale: "bad"}}
	code := cli.RunAdviseWithJudge(dir, rulesPath, j)
	if code != 0 {
		t.Errorf("expected exit 0 even when judge returns fail, got %d", code)
	}
	if j.calls != 1 {
		t.Errorf("expected judge to be called once, got %d", j.calls)
	}
}

func TestRunAdvise_JudgeError_ExitsZero(t *testing.T) {
	dir, rulesPath := makeMinimalRulesYAML(t, true)
	j := &mockJudge{err: errors.New("some error")}
	code := cli.RunAdviseWithJudge(dir, rulesPath, j)
	if code != 0 {
		t.Errorf("expected exit 0 even when judge errors, got %d", code)
	}
	if j.calls != 1 {
		t.Errorf("expected judge to be called once despite error, got %d", j.calls)
	}
}

func TestRunAdvise_NoSemanticRules_ExitsZero(t *testing.T) {
	dir, rulesPath := makeMinimalRulesYAML(t, false)
	j := &mockJudge{verdict: semantic.Verdict{Verdict: "pass", Rationale: "ok"}}
	code := cli.RunAdviseWithJudge(dir, rulesPath, j)
	if code != 0 {
		t.Errorf("expected exit 0, got %d", code)
	}
	// semantic エントリが無ければ judge は呼ばれない
	if j.calls != 0 {
		t.Errorf("expected judge not to be called, got %d", j.calls)
	}
}
