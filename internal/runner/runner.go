package runner

import (
	"context"
	"log/slog"
	"time"

	"github.com/haasonsaas/pve-oci-operator/internal/reconciler"
	"github.com/haasonsaas/pve-oci-operator/internal/spec"
)

type Runner struct {
	Reconciler  *reconciler.Reconciler
	ServicesDir string
	Interval    time.Duration
	Logger      *slog.Logger
}

func (r *Runner) Start(ctx context.Context) error {
	if r.Logger == nil {
		r.Logger = slog.Default()
	}
	if r.Interval == 0 {
		r.Interval = 10 * time.Second
	}
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()
	if err := r.runOnce(ctx); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.runOnce(ctx); err != nil {
				r.Logger.Error("reconcile tick failed", "error", err)
			}
		}
	}
}

func (r *Runner) runOnce(ctx context.Context) error {
	services, err := spec.LoadServiceSpecs(r.ServicesDir)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if err := r.Reconciler.Reconcile(ctx, svc); err != nil {
			r.Logger.Error("reconcile failed", "service", svc.Metadata.Name, "error", err)
		}
	}
	return nil
}
