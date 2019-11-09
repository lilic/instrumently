package metrics

import (
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func NewClientForGVK(cfg *rest.Config, apiVersion, kind string) (dynamic.NamespaceableResourceInterface, error) {
	apiResourceList, apiResource, err := getAPIResource(cfg, apiVersion, kind)
	if err != nil {
		return nil, errors.Wrapf(err, "discovering resource information failed for %s in %s", kind, apiVersion)
	}

	dc, err := newForConfig(cfg, apiResourceList.GroupVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "creating dynamic client failed for %s", apiResourceList.GroupVersion)
	}

	gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing GroupVersion %s failed", apiResourceList.GroupVersion)
	}

	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: apiResource.Name,
	}

	return dc.Resource(gvr), nil
}

func getAPIResource(cfg *rest.Config, apiVersion, kind string) (*metav1.APIResourceList, *metav1.APIResource, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	apiResourceLists, err := kclient.Discovery().ServerResources()
	if err != nil {
		return nil, nil, err
	}

	for _, apiResourceList := range apiResourceLists {
		if apiResourceList.GroupVersion == apiVersion {
			for _, r := range apiResourceList.APIResources {
				if r.Kind == kind {
					return apiResourceList, &r, nil
				}
			}
		}
	}

	return nil, nil, fmt.Errorf("apiVersion %s and kind %s not found available in Kubernetes cluster", apiVersion, kind)
}

func newForConfig(c *rest.Config, groupVersion string) (dynamic.Interface, error) {
	config := rest.CopyConfig(c)

	err := setConfigDefaults(groupVersion, config)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

func setConfigDefaults(groupVersion string, config *rest.Config) error {
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return err
	}
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	if config.GroupVersion.Group == "" && config.GroupVersion.Version == "v1" {
		config.APIPath = "/api"
	}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return nil
}
