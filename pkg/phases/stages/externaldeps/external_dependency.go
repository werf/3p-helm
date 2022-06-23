package externaldeps

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
)

func NewExternalDependency(name, resourceType, resourceName string) *ExternalDependency {
	return &ExternalDependency{
		Name:         name,
		ResourceType: resourceType,
		ResourceName: resourceName,
	}
}

type ExternalDependency struct {
	Name         string
	ResourceType string
	ResourceName string

	Namespace string
	Info      *resource.Info
}

func (d *ExternalDependency) GenerateInfo(gvkBuilder GVKBuilder, metaAccessor meta.MetadataAccessor) error {
	gvk, err := gvkBuilder.BuildFromResource(d.ResourceType)
	if err != nil {
		return fmt.Errorf("error building GroupVersionKind from resource type: %w", err)
	}

	object := unstructured.Unstructured{}

	object.SetGroupVersionKind(*gvk)
	object.SetName(d.ResourceName)
	object.SetNamespace(d.Namespace)

	d.Info = &resource.Info{
		Object:    &object,
		Name:      d.ResourceName,
		Namespace: d.Namespace,
	}

	return nil
}
