# instrumently

This repository implements a simple controller/operator for watching Foo resources as
defined with a CustomResourceDefinition (CRD) and is intrsumented with custom resource
metrics.

**Note:** go-get or vendor this package as `github.com/lilic/instrumently`.

## Where does it come from?

instrumently is the instruemnted `sample-controller` which is synced from
https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/sample-controller.
