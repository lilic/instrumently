package metrics

import (
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/common/log"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

func ServeMetrics(stores [][]*metricsstore.MetricsStore, host string, port int32) {
	listenAddress := net.JoinHostPort(host, fmt.Sprint(port))
	mux := http.NewServeMux()
	// Add metricsPath
	mux.Handle(metricsPath, &metricHandler{stores})
	err := http.ListenAndServe(listenAddress, mux)
	log.Error(err, "Failed to serve custom metrics")
}

type metricHandler struct {
	stores [][]*metricsstore.MetricsStore
}

func (m *metricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	// 0.0.4 is the exposition format version of prometheus
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")
	for _, stores := range m.stores {
		for _, s := range stores {
			s.WriteAll(w)
		}
	}
}
