package fake

import (
	"context"

	"github.com/werf/3p-helm-legacy/pkg/kube"
)

func (c *PrintingKubeClient) DeleteNamespace(ctx context.Context, namespace string, opts kube.DeleteOptions) error {
	return nil
}
