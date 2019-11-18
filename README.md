# instrumently

This repository implements a simple controller/operator for watching Foo resources as
defined with a CustomResourceDefinition (CRD) and is instrumented with custom resource
metrics.

**Note:** go-get or vendor this package as `github.com/lilic/instrumently`.

## Where does it come from?

instrumently is the instrumented fork of the `sample-controller` which is synced from
https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/sample-controller.


## How to use this in your operator?

Either import the [metrics package](/metrics) from this repository or copy over the files.

In the main package of your operator (main.go) paste the following lines of code:

```go
func serveOperatorMetrics(cfg *rest.Config) error {
	// Create new Unstructured client.
	client, err := metrics.NewClientForGVK(cfg, "samplecontroller.k8s.io/v1alpha1", "Foo")
	if err != nil {
		return err
	}
	// Generate collector in given namespace based on the Custom Resource API group/version, kind and the metrics.
	gvkStores := metrics.NewMetricsStores(client, []string{"default"}, "samplecontroller.k8s.io/v1alpha1", "Foo", metricFamilies)
	// Start serving metrics on local host on port 8383.
	go metrics.ServeMetrics([][]*metricsstore.MetricsStore{gvkStores}, "0.0.0.0", 8383)

	return nil
}
```

And call the function `serveOperatorMetrics(kubeConfig)` with `kubeConfig` from the `main()` function of your operator.

## How to register different metrics about your operator

Add more metrics to the `metricFamilies` variable at the top of the `main.go` file.

