// Package main is the pegasusX backend lifecycle entrypoint.
//
// main.go is a thin uber-go/fx composition root: initialize structured logging
// → assemble modular fx.Option graph (ObservabilityModule, BootstrapModule,
// WorkersModule, HTTPModule) → fx.New(...).Run(). Target ceiling: 200 lines.
// Domain logic lives in domain packages; URL mounts live in http_router.go and *routes packages.
package main

import (
	"log/slog"
	"os"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/logx"
	"go.uber.org/fx"
)

func main() {
	logger := logx.NewLogger(os.Stdout, slog.LevelInfo)
	slog.SetDefault(logger)

	fx.New(AppOptions(logger)...).Run()
}
