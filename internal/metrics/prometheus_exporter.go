package metrics

import (
	"fmt"
	"net/http"

	errors "github.com/pkg/errors"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
	"k8s.io/klog/v2"
)

const (
	metricsEndpoint = "metrics"
)

// serve creates the http handler for serving metrics requests
func servePrometheusExporter(metricsAddress string) error {
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{
			0.1, 0.2, 0.3, 0.4, 0.5, 1, 1.5, 2, 2.5, 3.0, 5.0, 10.0, 15.0, 30.0,
		}},
	)
	if err != nil {
		return errors.Wrap(err, "failed to register prometheus exporter")
	}

	klog.InfoS("Prometheus metrics server starting", "address", metricsAddress)
	http.HandleFunc(fmt.Sprintf("/%s", metricsEndpoint), exporter.ServeHTTP)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", metricsAddress), nil); err != nil {
		return errors.Wrap(err, "failed to register prometheus endpoint")
	}

	return nil
}
