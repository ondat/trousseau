package metrics

import (
	"context"

	"github.com/ondat/trousseau/pkg/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
	"k8s.io/klog/v2"
)

const (
	instrumentationName  = "vaultprovider"
	errorMessageKey      = "error_message"
	statusTypeKey        = "status"
	operationTypeKey     = "operation"
	providerKey          = "provider"
	kmsRequestMetricName = "kms_request"
	// ErrorStatusTypeValue sets status tag to "error"
	ErrorStatusTypeValue = "error"
	// SuccessStatusTypeValue sets status tag to "success"
	SuccessStatusTypeValue = "success"
	// EncryptOperationTypeValue sets operation tag to "encrypt"
	EncryptOperationTypeValue = "encrypt"
	// DecryptOperationTypeValue sets operation tag to "decrypt"
	DecryptOperationTypeValue = "decrypt"
	// GrpcOperationTypeValue sets operation tag to "grpc"
	GrpcOperationTypeValue = "grpc"
)

var kmsRequest metric.Float64ValueRecorder

type reporter struct {
	meter metric.Meter
}

// StatsReporter reports metrics
type StatsReporter interface {
	ReportRequest(ctx context.Context, provider, operationType, status string, duration float64, errors ...string)
}

// NewStatsReporter instantiates otel reporter
func NewStatsReporter() StatsReporter {
	klog.V(logger.Debug1).Info("Initialize new stats reporter...")

	meter := global.Meter(instrumentationName)

	kmsRequest = metric.Must(meter).NewFloat64ValueRecorder(kmsRequestMetricName, metric.WithDescription("Distribution of how long it took for an operation"))

	return &reporter{
		meter: meter,
	}
}

func (r *reporter) ReportRequest(ctx context.Context, provider, operationType, status string, duration float64, errors ...string) {
	labels := []attribute.KeyValue{
		attribute.String(providerKey, provider),
		attribute.String(operationTypeKey, operationType),
		attribute.String(statusTypeKey, status),
	}

	// Add errors
	if (status == ErrorStatusTypeValue) && len(errors) > 0 {
		for _, err := range errors {
			labels = append(labels, attribute.String(errorMessageKey, err))
		}
	}

	r.meter.RecordBatch(ctx,
		labels,
		kmsRequest.Measurement(duration),
	)
}
