package spec

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleYAML = `apiVersion: pve.haasonsaas/v1
kind: Service
metadata:
  name: composer-web
spec:
  node: hephaestus-2
  ctid: 160
  image: ghcr.io/haasonsaas/composer-web
  tag: main
  pullPolicy: digest
  resources:
    cores: 4
    memoryMB: 8192
  network:
    bridge: vmbr0
    ip: 192.168.4.160/24
    gw: 192.168.4.1
  mounts:
    - host: /srv/devdata/composer
      guest: /srv/composer
      options: rw
  healthCheck:
    type: http
    url: http://192.168.4.160:8080/healthz
    timeoutSeconds: 3
    intervalSeconds: 10
    healthyThreshold: 3
  rollout:
    strategy: recreate
    maxUnavailable: 1
    autoRollback: true
`

func TestParseServiceSpec(t *testing.T) {
	svc, err := ParseServiceSpec([]byte(sampleYAML))
	if err != nil {
		t.Fatalf("ParseServiceSpec error: %v", err)
	}
	if svc.Metadata.Name != "composer-web" {
		t.Fatalf("unexpected name %s", svc.Metadata.Name)
	}
	if svc.Spec.Rollout.Strategy != "recreate" {
		t.Fatalf("unexpected strategy %s", svc.Spec.Rollout.Strategy)
	}
}

func TestLoadServiceSpecs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "composer.yml")
	if err := os.WriteFile(path, []byte(sampleYAML), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	specs, err := LoadServiceSpecs(dir)
	if err != nil {
		t.Fatalf("LoadServiceSpecs error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec got %d", len(specs))
	}
}
