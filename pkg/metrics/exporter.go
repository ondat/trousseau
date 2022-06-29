package metrics

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"
)

const (
	prometheusExporter = "prometheus"
)

// Serve starts new exporter if needed
func Serve(metricsBackend, metricsAddress string) error {
	exporter := strings.ToLower(metricsBackend)
	klog.InfoS("Metrics backend", "exporter", exporter)

	switch exporter {
	// Prometheus is the only exporter supported for now
	case prometheusExporter:
		return servePrometheusExporter(metricsAddress)
	default:
		return fmt.Errorf("unsupported metrics backend %v", metricsBackend)
	}
}
