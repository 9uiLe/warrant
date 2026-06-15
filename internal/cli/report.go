package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/9uiLe/warrant/internal/check"
	"github.com/9uiLe/warrant/internal/registry"
	"github.com/9uiLe/warrant/internal/report"
)

func runReport(args []string) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	var root, cfgPath, regPath string
	fs.StringVar(&root, "repo-root", ".", "repo root directory")
	fs.StringVar(&cfgPath, "config", "", "config file path")
	fs.StringVar(&regPath, "registry", "", "registry file path")
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

	if err := report.Write(absRoot, cfg.ReportPath, result.Requirements, result.Violations); err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}

	fmt.Printf("レポートを書き込みました: %s\n", cfg.ReportPath)

	if len(result.Violations) > 0 {
		return 1
	}
	return 0
}
