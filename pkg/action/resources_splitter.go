package action

type ResourcesSplitter interface{
  Split(current, target kube.ResourceList) ([]kube.ResourceList, error)
}
