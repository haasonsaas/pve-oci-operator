package reconciler

import (
	"context"
	"testing"

	"github.com/evalops/pve-oci-operator/internal/pve"
	"github.com/evalops/pve-oci-operator/internal/spec"
)

type fakeRegistry struct {
	digest string
}

func (f *fakeRegistry) ResolveDigest(context.Context, string, string) (string, error) {
	return f.digest, nil
}

type fakePVE struct {
	actual pve.ActualState
	op     []string
}

func (f *fakePVE) GetContainer(context.Context, string, int) (pve.ActualState, error) {
	return f.actual, nil
}

func (f *fakePVE) CreateContainer(context.Context, spec.ServiceSpec, string) error {
	f.op = append(f.op, "create")
	f.actual.Exists = true
	return nil
}

func (f *fakePVE) StopContainer(context.Context, string, int) error {
	f.op = append(f.op, "stop")
	return nil
}

func (f *fakePVE) StartContainer(context.Context, string, int) error {
	f.op = append(f.op, "start")
	return nil
}

func (f *fakePVE) DestroyContainer(context.Context, string, int) error {
	f.op = append(f.op, "destroy")
	return nil
}

type fakeHealth struct{}

func (fakeHealth) Wait(context.Context, spec.ServiceSpec) error { return nil }

func TestReconcilerDeploysFreshService(t *testing.T) {
	svc := spec.ServiceSpec{}
	svc.Metadata.Name = "composer"
	svc.Spec.Node = "node1"
	svc.Spec.CTID = 160
	svc.Spec.Image = "ghcr.io/evalops/composer"
	svc.Spec.Tag = "main"
	svc.Spec.Rollout.Strategy = "recreate"

	rec := Reconciler{
		Registry: &fakeRegistry{digest: "sha256:123"},
		PVE:      &fakePVE{},
		Health:   fakeHealth{},
	}
	if err := rec.Reconcile(context.Background(), svc); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
}

func TestReconcilerRollsOutWhenDigestChanges(t *testing.T) {
	fpve := &fakePVE{actual: pve.ActualState{Exists: true, CurrentDigest: "sha256:old"}}
	svc := spec.ServiceSpec{}
	svc.Metadata.Name = "composer"
	svc.Spec.Node = "node1"
	svc.Spec.CTID = 160
	svc.Spec.Image = "ghcr.io/evalops/composer"
	svc.Spec.Tag = "main"
	svc.Spec.Rollout.Strategy = "recreate"
	svc.Spec.Rollout.AutoRollback = true

	rec := Reconciler{
		Registry: &fakeRegistry{digest: "sha256:new"},
		PVE:      fpve,
		Health:   fakeHealth{},
	}
	if err := rec.Reconcile(context.Background(), svc); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(fpve.op) == 0 {
		t.Fatalf("expected operations")
	}
}
