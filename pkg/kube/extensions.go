package kube

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacv1alpha1 "k8s.io/api/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/cli-runtime/pkg/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) DeleteNamespace(ctx context.Context, namespace string, opts DeleteOptions) error {
	cs, err := c.Factory.KubernetesClientSet()
	if err != nil {
		return err
	}

	if err := cs.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if opts.Wait {
		specs := []*ResourcesWaiterDeleteResourceSpec{
			{ResourceName: namespace, Namespace: "", GroupVersionResource: corev1.SchemeGroupVersion.WithResource("namespaces")},
		}
		if err := c.ResourcesWaiter.WaitUntilDeleted(context.Background(), specs, opts.WaitTimeout); err != nil {
			return fmt.Errorf("waiting until namespace deleted failed: %s", err)
		}
	}

	return nil
}

func (c *Client) keepResourcesSupportingDryRun(resources ResourceList) (ResourceList, error) {
	verifier, err := c.DryRunVerifierGetter.DryRunVerifier()
	if err != nil {
		return nil, fmt.Errorf("unable to get server dry run verifier: %w", err)
	}

	return resources.Filter(isDryRunSupportedFilter(verifier)), nil
}

func isDryRunSupportedFilter(dryRunVerifier *resource.DryRunVerifier) func(*resource.Info) bool {
	return func(info *resource.Info) bool {
		if err := dryRunVerifier.HasSupport(info.Mapping.GroupVersionKind); err != nil {
			return false
		}

		switch AsVersioned(info).(type) {
		case *rbacv1.ClusterRoleBinding, *rbacv1alpha1.ClusterRoleBinding, *rbacv1beta1.ClusterRoleBinding:
			return false
		}

		return true
	}
}

func isDryRunUnsupportedError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "admission webhook") && strings.HasSuffix(err.Error(), "does not support dry run")
}
