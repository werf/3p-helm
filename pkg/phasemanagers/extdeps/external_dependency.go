package extdeps

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
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

func (d *ExternalDependency) GenerateInfo(gvkBuilder GVKBuilder, scheme *runtime.Scheme, metaAccessor meta.MetadataAccessor) error {
	gvk, err := gvkBuilder.BuildFromResource(d.ResourceType)
	if err != nil {
		return fmt.Errorf("error building GroupVersionKind from resource type: %w", err)
	}

	object, err := scheme.New(*gvk)
	if err != nil {
		return fmt.Errorf("error creating new object from GroupVersionKind: %w", err)
	}

	object.GetObjectKind().SetGroupVersionKind(*gvk)

	if err := metaAccessor.SetName(object, d.ResourceName); err != nil {
		return fmt.Errorf("error setting name for an object: %w", err)
	}

	if d.Namespace != "" {
		if err := metaAccessor.SetNamespace(object, d.Namespace); err != nil {
			return fmt.Errorf("error setting namespace for an object: %w", err)
		}
	}

	d.Info = &resource.Info{
		Object:    object,
		Name:      d.ResourceName,
		Namespace: d.Namespace,
	}

	return nil
}
