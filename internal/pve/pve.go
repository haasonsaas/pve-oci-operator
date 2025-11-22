package pve

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/haasonsaas/pve-oci-operator/internal/spec"
	"github.com/haasonsaas/pve-oci-operator/internal/state"
)

type Client interface {
	GetContainer(ctx context.Context, node string, ctid int) (ActualState, error)
	CreateContainer(ctx context.Context, svc spec.ServiceSpec, digest string) error
	StopContainer(ctx context.Context, node string, ctid int) error
	StartContainer(ctx context.Context, node string, ctid int) error
	DestroyContainer(ctx context.Context, node string, ctid int) error
}

type ActualState struct {
	Exists        bool
	CTID          int
	Node          string
	CurrentDigest string
	Status        string
}

type CLIClient struct {
	pctPath string
	store   state.Store
	dryRun  bool
}

func NewCLIClient(pctPath string, store state.Store, dryRun bool) *CLIClient {
	return &CLIClient{pctPath: pctPath, store: store, dryRun: dryRun}
}

func (c *CLIClient) GetContainer(ctx context.Context, node string, ctid int) (ActualState, error) {
	actual := ActualState{CTID: ctid, Node: node}
	out, err := c.run(ctx, "status", strconv.Itoa(ctid))
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return actual, nil
		}
		return actual, err
	}
	actual.Exists = true
	actual.Status = parseStatus(out)
	entry, ok, err := c.store.Load(ctid)
	if err != nil {
		return actual, err
	}
	if ok {
		actual.CurrentDigest = entry.Digest
	}
	return actual, nil
}

func (c *CLIClient) CreateContainer(ctx context.Context, svc spec.ServiceSpec, digest string) error {
	if c.dryRun {
		return c.store.Save(state.Entry{CTID: svc.Spec.CTID, Digest: digest, Status: "running", Node: svc.Spec.Node})
	}
	args := []string{
		"create",
		strconv.Itoa(svc.Spec.CTID),
		fmt.Sprintf("%s@%s", svc.Spec.Image, digest),
		"--hostname", svc.Metadata.Name,
		"--cores", strconv.Itoa(svc.Spec.Resources.Cores),
		"--memory", strconv.Itoa(svc.Spec.Resources.MemoryMB),
	}
	if svc.Spec.Network.Bridge != "" {
		args = append(args, "--net0", fmt.Sprintf("name=%s,ip=%s,gw=%s", svc.Spec.Network.Bridge, svc.Spec.Network.IP, svc.Spec.Network.GW))
	}
	if err := c.exec(ctx, args...); err != nil {
		return err
	}
	return c.store.Save(state.Entry{CTID: svc.Spec.CTID, Digest: digest, Status: "stopped", Node: svc.Spec.Node})
}

func (c *CLIClient) StopContainer(ctx context.Context, _ string, ctid int) error {
	if c.dryRun {
		entry, ok, err := c.store.Load(ctid)
		if err != nil {
			return err
		}
		if ok {
			entry.Status = "stopped"
			return c.store.Save(entry)
		}
		return nil
	}
	return c.exec(ctx, "stop", strconv.Itoa(ctid))
}

func (c *CLIClient) StartContainer(ctx context.Context, _ string, ctid int) error {
	if c.dryRun {
		entry, ok, err := c.store.Load(ctid)
		if err != nil {
			return err
		}
		if ok {
			entry.Status = "running"
			return c.store.Save(entry)
		}
		return nil
	}
	return c.exec(ctx, "start", strconv.Itoa(ctid))
}

func (c *CLIClient) DestroyContainer(ctx context.Context, _ string, ctid int) error {
	if c.dryRun {
		return c.store.Remove(ctid)
	}
	return c.exec(ctx, "destroy", strconv.Itoa(ctid))
}

func (c *CLIClient) exec(ctx context.Context, args ...string) error {
	_, err := c.run(ctx, args...)
	return err
}

func (c *CLIClient) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.pctPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pct %v: %w: %s", args, err, out)
	}
	return string(out), nil
}

func parseStatus(out string) string {
	parts := strings.Split(strings.TrimSpace(out), ":")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(out)
}
