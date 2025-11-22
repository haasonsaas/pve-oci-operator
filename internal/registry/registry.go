package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Client resolves OCI image references into immutable digests.
type Client interface {
	ResolveDigest(ctx context.Context, image, tag string) (string, error)
}

type OCIClient struct {
	auth authn.Authenticator
}

func NewOCIClient(username, password string) *OCIClient {
	var auth authn.Authenticator = authn.Anonymous
	if username != "" || password != "" {
		auth = &authn.Basic{Username: username, Password: password}
	}
	return &OCIClient{auth: auth}
}

func (c *OCIClient) ResolveDigest(ctx context.Context, image, tag string) (string, error) {
	refStr := buildReference(image, tag)
	ref, err := name.ParseReference(refStr)
	if err != nil {
		return "", fmt.Errorf("parse reference %s: %w", refStr, err)
	}
	desc, err := remote.Head(ref, remote.WithContext(ctx), remote.WithAuth(c.auth))
	if err != nil {
		return "", fmt.Errorf("resolve digest for %s: %w", refStr, err)
	}
	return desc.Digest.String(), nil
}

func buildReference(image, tag string) string {
	if strings.HasPrefix(tag, "sha256:") {
		return fmt.Sprintf("%s@%s", image, tag)
	}
	if strings.HasPrefix(tag, "@sha256:") {
		return fmt.Sprintf("%s%s", image, tag)
	}
	return fmt.Sprintf("%s:%s", image, tag)
}
