package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/9uiLe/warrant/internal/registry"
	"github.com/9uiLe/warrant/internal/serve"
)

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	var root, cfgPath, regPath string
	var port int
	fs.StringVar(&root, "repo-root", ".", "repo root directory")
	fs.StringVar(&cfgPath, "config", "", "config file path")
	fs.StringVar(&regPath, "registry", "", "registry file path")
	fs.IntVar(&port, "port", 7777, "port to listen on")
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

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("warrant serve: http://%s\n", addr)

	if err := serve.Start(addr, absRoot, reg, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "実行エラー:", err)
		return 2
	}
	return 0
}
