package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Main はサブコマンドを dispatch し、終了コードを返す
// 終了コード: 0=PASS, 1=FAIL(violations), 2=実行エラー
func Main(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: warrant <command> [options]")
		fmt.Fprintln(os.Stderr, "commands: check, report, serve, init, advise, ratify, version")
		return 2
	}

	switch args[0] {
	case "check":
		return runCheck(args[1:])
	case "report":
		return runReport(args[1:])
	case "serve":
		return runServe(args[1:])
	case "init":
		return runInit(args[1:])
	case "advise":
		return runAdvise(args[1:])
	case "ratify":
		return runRatify(args[1:])
	case "version":
		return runVersion(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

// parseCommonFlags は共通フラグを定義してパースする
func parseCommonFlags(args []string, name string) (root, configPath, registryPath string, rest []string, err error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.StringVar(&root, "repo-root", ".", "repo root directory")
	var cfg, reg string
	fs.StringVar(&cfg, "config", "", "config file path")
	fs.StringVar(&reg, "registry", "", "registry file path")
	if err = fs.Parse(args); err != nil {
		return
	}
	if cfg == "" {
		cfg = filepath.Join(root, ".warrant", "config.yaml")
	}
	if reg == "" {
		reg = filepath.Join(root, ".warrant", "requirements.yaml")
	}
	configPath = cfg
	registryPath = reg
	rest = fs.Args()
	return
}
