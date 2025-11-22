package reconciler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/haasonsaas/pve-oci-operator/internal/health"
	"github.com/haasonsaas/pve-oci-operator/internal/pve"
	"github.com/haasonsaas/pve-oci-operator/internal/registry"
	"github.com/haasonsaas/pve-oci-operator/internal/spec"
)

type Reconciler struct {
	Registry registry.Client
	PVE      pve.Client
	Health   health.Checker
	Logger   *slog.Logger
}

func (r *Reconciler) Reconcile(ctx context.Context, svc spec.ServiceSpec) error {
	if r.Logger == nil {
		r.Logger = slog.Default()
	}
	digest, err := r.resolveDigest(ctx, svc)
	if err != nil {
		return err
	}
	actual, err := r.PVE.GetContainer(ctx, svc.Spec.Node, svc.Spec.CTID)
	if err != nil {
		return err
	}
	if !actual.Exists {
		r.Logger.Info("deploying", "service", svc.Metadata.Name, "digest", digest)
		return r.deployFresh(ctx, svc, digest)
	}
	if actual.CurrentDigest == digest {
		r.Logger.Info("up to date", "service", svc.Metadata.Name, "digest", digest)
		return nil
	}
	r.Logger.Info("rollout required", "service", svc.Metadata.Name, "from", actual.CurrentDigest, "to", digest)
	switch strings.ToLower(svc.Spec.Rollout.Strategy) {
	case "recreate":
		return r.recreate(ctx, svc, actual, digest)
	case "bluegreen":
		return r.blueGreen(ctx, svc, actual, digest)
	default:
		return fmt.Errorf("unknown rollout strategy %q", svc.Spec.Rollout.Strategy)
	}
}

func (r *Reconciler) resolveDigest(ctx context.Context, svc spec.ServiceSpec) (string, error) {
	policy := strings.ToLower(svc.Spec.PullPolicy)
	if strings.HasPrefix(svc.Spec.Tag, "sha256:") || policy == "never" {
		return svc.Spec.Tag, nil
	}
	if policy == "digest" || policy == "" || policy == "tag" {
		return r.Registry.ResolveDigest(ctx, svc.Spec.Image, svc.Spec.Tag)
	}
	return "", fmt.Errorf("unsupported pullPolicy %s", svc.Spec.PullPolicy)
}

func (r *Reconciler) deployFresh(ctx context.Context, svc spec.ServiceSpec, digest string) error {
	if err := r.PVE.CreateContainer(ctx, svc, digest); err != nil {
		return err
	}
	if err := r.PVE.StartContainer(ctx, svc.Spec.Node, svc.Spec.CTID); err != nil {
		return err
	}
	return r.Health.Wait(ctx, svc)
}

func (r *Reconciler) recreate(ctx context.Context, svc spec.ServiceSpec, actual pve.ActualState, digest string) error {
	prevDigest := actual.CurrentDigest
	if err := r.PVE.StopContainer(ctx, svc.Spec.Node, svc.Spec.CTID); err != nil {
		return err
	}
	if err := r.PVE.DestroyContainer(ctx, svc.Spec.Node, svc.Spec.CTID); err != nil {
		return err
	}
	if err := r.deployFresh(ctx, svc, digest); err != nil {
		if svc.Spec.Rollout.AutoRollback && prevDigest != "" {
			r.Logger.Error("rollout failed, attempting rollback", "service", svc.Metadata.Name, "error", err)
			return errors.Join(err, r.rollback(ctx, svc, prevDigest))
		}
		return err
	}
	return nil
}

func (r *Reconciler) rollback(ctx context.Context, svc spec.ServiceSpec, digest string) error {
	if err := r.PVE.DestroyContainer(ctx, svc.Spec.Node, svc.Spec.CTID); err != nil {
		return err
	}
	return r.deployFresh(ctx, svc, digest)
}

func (r *Reconciler) blueGreen(ctx context.Context, _ spec.ServiceSpec, _ pve.ActualState, _ string) error {
	return fmt.Errorf("blueGreen rollout not implemented yet")
}
