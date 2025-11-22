package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/haasonsaas/pve-oci-operator/internal/spec"
)

type Checker interface {
	Wait(ctx context.Context, svc spec.ServiceSpec) error
}

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{client: &http.Client{}}
}

func (h *HTTPChecker) Wait(ctx context.Context, svc spec.ServiceSpec) error {
	cfg := svc.Spec.Health
	if cfg.Type != "http" || cfg.URL == "" {
		return nil
	}
	interval := durationOrDefault(cfg.IntervalSeconds, 10)
	timeout := durationOrDefault(cfg.TimeoutSeconds, 3)
	required := cfg.HealthyThreshold
	if required <= 0 {
		required = 1
	}
	healthy := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := h.ping(ctx, cfg.URL, timeout); err != nil {
			healthy = 0
		} else {
			healthy++
			if healthy >= required {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (h *HTTPChecker) ping(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("health check status %d", resp.StatusCode)
}

func durationOrDefault(value int, fallback int) time.Duration {
	if value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}
