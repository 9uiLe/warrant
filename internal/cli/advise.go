// @warrant-covers WARRANT-ADVISE
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/9uiLe/warrant/internal/authority"
	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/glob"
	"github.com/9uiLe/warrant/internal/semantic"
)

const adviseWarning = "これは advisory であり CI ゲートではない。判断根拠にするには人間の承認が必要です (ref: P21/P17)"

func runAdvise(args []string) int {
	fs := flag.NewFlagSet("advise", flag.ContinueOnError)
	rootFlag := fs.String("repo-root", ".", "repo root directory")
	rulesPath := fs.String("rules", "", "rules.yaml path")
	cfgPath := fs.String("config", "", "config file path")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root := *rootFlag
	if *rulesPath == "" {
		*rulesPath = filepath.Join(root, ".warrant", "rules.yaml")
	}
	if *cfgPath == "" {
		*cfgPath = filepath.Join(root, ".warrant", "config.yaml")
	}

	var cfg *config.Config
	if c, err := config.Load(*cfgPath); err == nil {
		cfg = c
	} else {
		cfg = config.Default()
	}

	judge, err := semantic.NewExecJudge(cfg.SemanticCommand, cfg.SemanticTimeoutSec)
	if err != nil {
		fmt.Fprintf(os.Stdout, "semantic judge 未設定: %s。advisory はスキップします\n", err.Error())
		return 0
	}

	return runAdviseWithJudgeInternal(root, *rulesPath, judge, *jsonOut)
}

// RunAdviseWithJudge はテスト注入用エントリポイント (jsonOut は常に false)
func RunAdviseWithJudge(root, rulesPath string, judge semantic.Judge) int {
	return runAdviseWithJudgeInternal(root, rulesPath, judge, false)
}

func runAdviseWithJudgeInternal(root, rulesPath string, judge semantic.Judge, jsonOut bool) int {
	type result struct {
		RuleID            string `json:"rule_id"`
		Criterion         string `json:"criterion"`
		Verdict           string `json:"verdict"`
		Rationale         string `json:"rationale"`
		ProposedAssertion string `json:"proposed_assertion,omitempty"`
	}

	// judge が nil なら semantic judge 未設定として skip
	if judge == nil {
		fmt.Fprintf(os.Stdout, "semantic judge 未設定: %s。advisory はスキップします\n", semantic.ErrNoCommand.Error())
		return 0
	}

	rules, err := loadRulesForAdvise(root, rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rules.yaml 読み込みエラー: %v\n", err)
		return 0
	}

	var results []result

	for _, rule := range rules {
		for _, entry := range rule.EnforcedBy {
			if entry.Kind != "semantic" {
				continue
			}

			// governs グロブ展開
			var targets []string
			for _, pattern := range entry.Governs {
				matched, err := glob.Glob(root, pattern)
				if err != nil {
					continue
				}
				targets = append(targets, matched...)
			}

			req := semantic.Request{
				RuleID:    rule.ID,
				Title:     rule.Title,
				Basis:     rule.Basis,
				Criterion: entry.Criterion,
				Targets:   targets,
			}

			v, err := judge.Judge(context.Background(), req)
			if err != nil {
				results = append(results, result{
					RuleID:    rule.ID,
					Criterion: entry.Criterion,
					Verdict:   "error",
					Rationale: fmt.Sprintf("judge 失敗: %s", err.Error()),
				})
				continue
			}

			results = append(results, result{
				RuleID:            rule.ID,
				Criterion:         entry.Criterion,
				Verdict:           v.Verdict,
				Rationale:         v.Rationale,
				ProposedAssertion: v.ProposedAssertion,
			})
		}
	}

	if jsonOut {
		type jsonOutput struct {
			Advisory bool     `json:"advisory"`
			Warning  string   `json:"warning"`
			Results  []result `json:"results"`
		}
		out := jsonOutput{
			Advisory: true,
			Warning:  adviseWarning,
			Results:  results,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
		return 0
	}

	fmt.Fprintf(os.Stdout, "[advisory] %s\n", adviseWarning)
	for _, r := range results {
		fmt.Fprintf(os.Stdout, "\n[%s]\n", r.RuleID)
		fmt.Fprintf(os.Stdout, "  criterion: %s\n", r.Criterion)
		fmt.Fprintf(os.Stdout, "  verdict:   %s\n", r.Verdict)
		fmt.Fprintf(os.Stdout, "  rationale: %s\n", r.Rationale)
		if r.ProposedAssertion != "" {
			fmt.Fprintf(os.Stdout, "  proposed_assertion: %s\n", r.ProposedAssertion)
		}
	}
	return 0
}

func loadRulesForAdvise(root, rulesPath string) ([]authority.Rule, error) {
	cfg := config.Default()
	result, err := authority.Run(root, rulesPath, cfg)
	if err != nil {
		return nil, err
	}
	return result.Rules, nil
}
