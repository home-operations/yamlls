package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/home-operations/yamlls/internal/lsp"
	"github.com/home-operations/yamlls/internal/render"
	"github.com/home-operations/yamlls/internal/render/flate"
	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp/server"
)

var version = "0.0.0-dev"

func main() {
	var (
		showVersion bool
		logFile     string
		verbosity   int
	)
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&logFile, "log-file", "", "write log output to this file instead of stderr")
	flag.IntVar(&verbosity, "v", 0, "log verbosity (0=silent, 1=info, 2+=debug)")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		return
	}

	var logPath *string
	if logFile != "" {
		logPath = &logFile
	}
	commonlog.Configure(verbosity, logPath)

	registry := render.NewRegistry()
	registry.Register(flate.New())

	s := lsp.New(version, registry)
	srv := server.NewServer(s.Handler(), "yamlls", false)
	if err := srv.RunStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "server stopped with error: %v\n", err)
		os.Exit(1)
	}
}
