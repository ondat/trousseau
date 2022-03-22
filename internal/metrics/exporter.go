package metrics

import (
	"strings"

	errors "github.com/pkg/errors"
	"k8s.io/klog/v2"
)

const (
	prometheusExporter = "prometheus"
)

// InitMetricsExporter initializes new exporter
func InitMetricsExporter(metricsBackend, metricsAddress string) error {
	exporter := strings.ToLower(metricsBackend)
	klog.Infof("metrics backend: %s", exporter)

	switch exporter {
	// Prometheus is the only exporter supported for now
	case prometheusExporter:
		return initPrometheusExporter(metricsAddress)
	default:
		return errors.Errorf("unsupported metrics backend %v", metricsBackend)
	}
}
