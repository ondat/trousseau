package metrics

import (
	"fmt"
	"net/http"
	"time"

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
		return fmt.Errorf("failed to register prometheus exporter: %w", err)
	}

	klog.InfoS("Prometheus metrics server starting", "address", metricsAddress)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(fmt.Sprintf("/%s", metricsEndpoint), exporter.ServeHTTP)

	const timeout = time.Second * 5

	srv := &http.Server{
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout,
		Addr:              fmt.Sprintf(":%s", metricsAddress),
		Handler:           serveMux,
	}

	if err := srv.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to register prometheus endpoint: %w", err)
	}

	return nil
}
