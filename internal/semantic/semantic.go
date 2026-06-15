// @warrant-covers WARRANT-ADVISE
package semantic

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Request は外部 Judge コマンドへ渡す JSON ペイロード
type Request struct {
	RuleID    string   `json:"rule_id"`
	Title     string   `json:"title"`
	Basis     string   `json:"basis"`
	Criterion string   `json:"criterion"`
	Targets   []string `json:"targets"`
}

// Verdict は外部 Judge コマンドから受け取る JSON ペイロード
type Verdict struct {
	Verdict           string `json:"verdict"` // "pass" | "fail" | "uncertain"
	Rationale         string `json:"rationale"`
	ProposedAssertion string `json:"proposed_assertion,omitempty"`
}

// Judge は意味判定インターフェース
type Judge interface {
	Judge(ctx context.Context, req Request) (Verdict, error)
}

// ErrNoCommand は semantic_command 未設定のエラー
var ErrNoCommand = fmt.Errorf("semantic_command が設定されていない")

// ExecJudge は外部コマンドを起動して判定する Judge 実装
type ExecJudge struct {
	Command    string
	TimeoutSec int
}

// NewExecJudge は semantic_command が空でなければ ExecJudge を、空なら nil と ErrNoCommand を返す
func NewExecJudge(command string, timeoutSec int) (*ExecJudge, error) {
	if command == "" {
		return nil, ErrNoCommand
	}
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	return &ExecJudge{Command: command, TimeoutSec: timeoutSec}, nil
}

// Judge は外部コマンドを起動し Request を stdin で渡し stdout から Verdict を受け取る
func (e *ExecJudge) Judge(ctx context.Context, req Request) (Verdict, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSec)*time.Second)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return Verdict{}, fmt.Errorf("request marshal: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", e.Command)
	cmd.Stdin = strings.NewReader(string(payload))
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Verdict{}, fmt.Errorf("judge タイムアウト(%d秒): %w", e.TimeoutSec, ctx.Err())
		}
		return Verdict{}, fmt.Errorf("judge コマンド失敗: %w", err)
	}

	var v Verdict
	if err := json.Unmarshal(out, &v); err != nil {
		return Verdict{}, fmt.Errorf("verdict JSON デコード失敗: %w", err)
	}
	return v, nil
}
