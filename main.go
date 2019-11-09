/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/lilic/instrumently/metrics"
	clientset "github.com/lilic/instrumently/pkg/generated/clientset/versioned"
	informers "github.com/lilic/instrumently/pkg/generated/informers/externalversions"
	"github.com/lilic/instrumently/pkg/signals"

	"k8s.io/kube-state-metrics/pkg/metric"
	ksm "k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

var (
	masterURL  string
	kubeconfig string

	metricFamilies = []ksm.FamilyGenerator{
		ksm.FamilyGenerator{
			Name: "sample_controller_created",
			Type: ksm.Gauge,
			Help: "Unix creation timestamp for sample-controller",
			GenerateFunc: func(obj interface{}) *ksm.Family {
				cr := obj.(*unstructured.Unstructured)
				ms := []*ksm.Metric{}
				timestamp := cr.GetCreationTimestamp()
				if !timestamp.IsZero() {
					ms = append(ms, &ksm.Metric{
						Value:       float64(timestamp.Unix()),
						LabelKeys:   []string{"namespace", "name", "version"},
						LabelValues: []string{cr.GetNamespace(), cr.GetName(), cr.GetAPIVersion()},
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			},
		},
		ksm.FamilyGenerator{
			Name: "sample_controller_replicas",
			Type: ksm.Gauge,
			Help: "Number of replicas for sample-controller",
			GenerateFunc: func(obj interface{}) *ksm.Family {
				cr := obj.(*unstructured.Unstructured)
				content := cr.UnstructuredContent()
				spec := content["spec"].(map[string]interface{})
				replicas := spec["replicas"].(int64)
				return &metric.Family{
					Metrics: []*ksm.Metric{
						{
							Value:       float64(replicas),
							LabelKeys:   []string{"namespace", "name", "version"},
							LabelValues: []string{cr.GetNamespace(), cr.GetName(), cr.GetAPIVersion()},
						},
					},
				}
			},
		},
	}
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	exampleClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	exampleInformerFactory := informers.NewSharedInformerFactory(exampleClient, time.Second*30)

	controller := NewController(kubeClient, exampleClient,
		kubeInformerFactory.Apps().V1().Deployments(),
		exampleInformerFactory.Samplecontroller().V1alpha1().Foos())

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	exampleInformerFactory.Start(stopCh)

	err = serveOperatorMetrics(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}

}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func serveOperatorMetrics(cfg *rest.Config) error {
	// Create new Unstructured client.
	client, err := metrics.NewClientForGVK(cfg, "samplecontroller.k8s.io/v1alpha1", "Foo")
	if err != nil {
		return err
	}
	// Generate collector in given namespace based on the Custom Resource API group/version, kind and the metrics.
	gvkStores := metrics.NewMetricsStores(client, []string{"default"}, "samplecontroller.k8s.io/v1alpha1", "Foo", metricFamilies)
	// Start serving metrics on port 8383.
	go metrics.ServeMetrics([][]*metricsstore.MetricsStore{gvkStores}, "0.0.0.0", 8383)

	return nil
}
