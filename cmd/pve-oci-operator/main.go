package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/evalops/pve-oci-operator/internal/config"
	"github.com/evalops/pve-oci-operator/internal/health"
	"github.com/evalops/pve-oci-operator/internal/pve"
	"github.com/evalops/pve-oci-operator/internal/reconciler"
	"github.com/evalops/pve-oci-operator/internal/registry"
	"github.com/evalops/pve-oci-operator/internal/runner"
	"github.com/evalops/pve-oci-operator/internal/state"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to operator config")
	flag.Parse()

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	statePath := cfg.PVE.StatePath
	if statePath == "" {
		statePath = ".state"
	}
	store, err := state.NewFileStore(statePath)
	if err != nil {
		log.Fatalf("init state store: %v", err)
	}
	pveClient := pve.NewCLIClient(cfg.PVE.PctPath, store, cfg.PVE.DryRun)
	registryClient := registry.NewOCIClient(cfg.Registry.Username, cfg.Registry.Password)
	healthChecker := health.NewHTTPChecker()
	rec := &reconciler.Reconciler{Registry: registryClient, PVE: pveClient, Health: healthChecker, Logger: logger}
	run := &runner.Runner{Reconciler: rec, ServicesDir: cfg.Runner.ServicesPath, Interval: cfg.Runner.Interval, Logger: logger}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run.Start(ctx); err != nil && err != context.Canceled {
		logger.Error("runner stopped", "error", err)
	}
}
