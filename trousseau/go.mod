module github.com/ondat/trousseau/trousseau

go 1.18

replace (
	github.com/ondat/trousseau => ../

	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.7.0
)

require (
	github.com/ondat/trousseau v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.8.0
	google.golang.org/grpc v1.47.0
	k8s.io/apiserver v0.25.5
	k8s.io/klog/v2 v2.70.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	go.opentelemetry.io/otel v1.7.0 // indirect
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.20.0 // indirect
	go.opentelemetry.io/otel/internal/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/sdk v1.0.0-RC1 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.21.0 // indirect
	go.opentelemetry.io/otel/trace v1.7.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/net v0.3.1-0.20221206200815-1e63c2f08a10 // indirect
	golang.org/x/sys v0.3.0 // indirect
	golang.org/x/text v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20220519153652-3a47de7e79bd // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
