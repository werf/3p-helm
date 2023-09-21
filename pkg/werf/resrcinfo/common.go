package resrcinfo

import (
	"context"
	"fmt"

	"helm.sh/helm/v3/pkg/werf/kubeclnt"
	"helm.sh/helm/v3/pkg/werf/log"
	"helm.sh/helm/v3/pkg/werf/resrc"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
)

type UpToDateStatus string

const (
	UpToDateStatusUnknown UpToDateStatus = "unknown"
	UpToDateStatusYes     UpToDateStatus = "yes"
	UpToDateStatusNo      UpToDateStatus = "no"
)

func fixManagedFields(ctx context.Context, namespace string, getObj *unstructured.Unstructured, getResource *resrc.RemoteResource, kubeClient kubeclnt.KubeClienter, mapper meta.ResettableRESTMapper) error {
	if changed, err := getResource.FixManagedFields(); err != nil {
		return fmt.Errorf("error fixing managed fields: %w", err)
	} else if !changed {
		return nil
	}

	unstruct := unstructured.Unstructured{Object: map[string]interface{}{}}
	unstruct.SetManagedFields(getResource.Unstructured().GetManagedFields())

	patch, err := json.Marshal(unstruct.UnstructuredContent())
	if err != nil {
		return fmt.Errorf("error marshaling fixed managed fields: %w", err)
	}

	log.Default.Info(ctx, "Fixing managed fields for resource %q", getResource.HumanID())
	getObj, err = kubeClient.MergePatch(ctx, getResource.ResourceID, patch)
	if err != nil {
		return fmt.Errorf("error patching managed fields: %w", err)
	}

	getResource = resrc.NewRemoteResource(getObj, resrc.RemoteResourceOptions{
		FallbackNamespace: namespace,
		Mapper:            mapper,
	})

	return nil
}
