package driver

import (
    "fmt"
    "testing"

    rspb "k8s.io/helm/pkg/proto/hapi/release"
    "k8s.io/kubernetes/pkg/client/unversioned"
    kberrs "k8s.io/kubernetes/pkg/api/errors"
    "k8s.io/kubernetes/pkg/api"
)

func releaseStub(name string, vers int32, code rspb.Status_Code) *rspb.Release {
    return &rspb.Release{
        Name:    name,
        Version: vers,
        Info:    &rspb.Info{Status: &rspb.Status{Code: code}},
    }
}

func testKey(name string, vers int32) string {
    return fmt.Sprintf("%s.v%d", name, vers)
}

func tsFixtureMemory(t *testing.T) *Memory {
    hs := []*rspb.Release{
        // rls-a
        releaseStub("rls-a", 4, rspb.Status_DEPLOYED),
        releaseStub("rls-a", 1, rspb.Status_SUPERSEDED),
        releaseStub("rls-a", 3, rspb.Status_SUPERSEDED),
        releaseStub("rls-a", 2, rspb.Status_SUPERSEDED),
        // rls-b
        releaseStub("rls-b", 4, rspb.Status_DEPLOYED),
        releaseStub("rls-b", 1, rspb.Status_SUPERSEDED),
        releaseStub("rls-b", 3, rspb.Status_SUPERSEDED),
        releaseStub("rls-b", 2, rspb.Status_SUPERSEDED),
    }

    mem := NewMemory()
    for _, tt := range hs {
        err := mem.Create(testKey(tt.Name, tt.Version), tt)
        if err != nil {
            t.Fatalf("Test setup failed to create: %s\n", err)
        }
    }
    return mem
}

// newTestFixture initializes a MockConfigMapsInterface.
// ConfigMaps are created for each release provided.
func newTestFixtureCfgMaps(t *testing.T, releases ...*rspb.Release) *ConfigMaps {
    var mock MockConfigMapsInterface
    mock.Init(t, releases...)

    return NewConfigMaps(&mock)
}

// MockConfigMapsInterface mocks a kubernetes ConfigMapsInterface
type MockConfigMapsInterface struct {
    unversioned.ConfigMapsInterface

    objects map[string]*api.ConfigMap
}

func (mock *MockConfigMapsInterface) Init(t *testing.T, releases ...*rspb.Release) {
    mock.objects = map[string]*api.ConfigMap{}

    for _, rls := range releases {
        objkey := testKey(rls.Name, rls.Version)

        cfgmap, err := newConfigMapsObject(objkey, rls, nil)
        if err != nil {
            t.Fatalf("Failed to create configmap: %s", err)
        }
        mock.objects[objkey] = cfgmap
    }
}

func (mock *MockConfigMapsInterface) Get(name string) (*api.ConfigMap, error) {
    object, ok := mock.objects[name]
    if !ok {
        return nil, kberrs.NewNotFound(api.Resource("tests"), name)
    }
    return object, nil
}

func (mock *MockConfigMapsInterface) List(opts api.ListOptions) (*api.ConfigMapList, error) {
    var list api.ConfigMapList
    for _, cfgmap := range mock.objects {
        list.Items = append(list.Items, *cfgmap)
    }
    return &list, nil
}

func (mock *MockConfigMapsInterface) Create(cfgmap *api.ConfigMap) (*api.ConfigMap, error) {
    name := cfgmap.ObjectMeta.Name
    if object, ok := mock.objects[name]; ok {
        return object, kberrs.NewAlreadyExists(api.Resource("tests"), name)
    }
    mock.objects[name] = cfgmap
    return cfgmap, nil
}

func (mock *MockConfigMapsInterface) Update(cfgmap *api.ConfigMap) (*api.ConfigMap, error) {
    name := cfgmap.ObjectMeta.Name
    if _, ok := mock.objects[name]; !ok {
        return nil, kberrs.NewNotFound(api.Resource("tests"), name)
    }
    mock.objects[name] = cfgmap
    return cfgmap, nil
}

func (mock *MockConfigMapsInterface) Delete(name string) error {
    if _, ok := mock.objects[name]; !ok {
        return kberrs.NewNotFound(api.Resource("tests"), name)
    }
    delete(mock.objects, name)
    return nil
}
