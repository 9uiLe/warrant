package main

import (
	"os"

	"github.com/9uiLe/warrant/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
