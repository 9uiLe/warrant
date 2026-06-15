package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/config"
	"github.com/9uiLe/warrant/internal/registry"
)

func runCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	var root, cfgPath, regPath string
	var jsonOut bool
	fs.StringVar(&root, "repo-root", ".", "repo root directory")
	fs.StringVar(&cfgPath, "config", "", "config file path")
	fs.StringVar(&regPath, "registry", "", "registry file path")
	fs.BoolVar(&jsonOut, "json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}
	if cfgPath == "" {
		cfgPath = filepath.Join(absRoot, ".warrant", "config.yaml")
	}
	if regPath == "" {
		regPath = filepath.Join(absRoot, ".warrant", "requirements.yaml")
	}

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	reg, err := registry.Load(regPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	result, err := check.Run(absRoot, reg, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	n := len(result.Requirements)
	vs := result.Violations

	if jsonOut {
		type checkJSONOut struct {
			OK           bool          `json:"ok"`
			Requirements int           `json:"requirements"`
			Violations   []interface{} `json:"violations"`
		}
		viols := make([]interface{}, len(vs))
		for i, v := range vs {
			viols[i] = v
		}
		out := checkJSONOut{
			OK:           len(vs) == 0,
			Requirements: n,
			Violations:   viols,
		}
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, "実行エラー:", err)
			return 2
		}
		fmt.Print(buf.String())
	} else {
		if len(vs) == 0 {
			fmt.Printf("OK: 機能 %d 件すべてが仕様とテストに機械可読に束縛されている（司法ゲート PASS）\n", n)
		} else {
			fmt.Fprintf(os.Stderr, "NG: %d 件の違反（司法ゲート FAIL）。機能 %d 件中。\n", len(vs), n)
			for _, v := range vs {
				fmt.Fprintf(os.Stderr, "  [%s] %s: %s\n", v.Code, v.Requirement, v.Message)
			}
		}
	}

	if len(vs) > 0 {
		return 1
	}
	return 0
}

func loadConfig(cfgPath string) (*config.Config, error) {
	_, err := os.Stat(cfgPath)
	if os.IsNotExist(err) {
		return config.Default(), nil
	}
	return config.Load(cfgPath)
}
