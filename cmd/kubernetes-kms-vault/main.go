package main

import (
	"context"
	"flag"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Trousseau-io/trousseau-tsh/internal/config"
	"github.com/Trousseau-io/trousseau-tsh/internal/metrics"
	"github.com/Trousseau-io/trousseau-tsh/internal/server"
	"github.com/Trousseau-io/trousseau-tsh/internal/utils"
	"github.com/Trousseau-io/trousseau-tsh/internal/version"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
)

var (
	listenAddr     = flag.String("listen-addr", "unix:///opt/vaultkms.socket", "gRPC listen address")
	logFormatJSON  = flag.Bool("log-format-json", false, "set log formatter to json")
	configFilePath = flag.String("config-file-path", "./config.yaml", "Path for Vault Provider config file")
	healthzPort    = flag.Int("healthz-port", 8787, "port for health check")
	healthzPath    = flag.String("healthz-path", "/healthz", "path for health check")
	healthzTimeout = flag.Duration("healthz-timeout", 20*time.Second, "RPC timeout for health check")
	metricsBackend = flag.String("metrics-backend", "prometheus", "Backend used for metrics")
	metricsAddress = flag.String("metrics-addr", "8095", "The address the metric endpoint binds to")
)

func main() {
	klog.InitFlags(nil)

	flag.Parse()
	if *logFormatJSON {
		klog.SetLogger(json.JSONLogger)
	}
	ctx := withShutdownSignal(context.Background())
	// initialize metrics exporter
	err := metrics.InitMetricsExporter(*metricsBackend, *metricsAddress)
	if err != nil {
		klog.Fatalf("failed to initialize metrics exporter, error: %+v", err)
	}
	klog.InfoS("Starting VaultEncryptionServiceServer service", "version", version.BuildVersion, "buildDate", version.BuildDate)
	cfg, err := config.New(configFilePath)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}
	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}
	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	kmsServer, err := server.New(ctx, cfg)
	pb.RegisterKeyManagementServiceServer(s, kmsServer)
	if err != nil {
		klog.Fatalf("failed to listen: %v", err)
	}
	klog.Infof("Listening for connections on address: %v", listener.Addr())
	go func() {
		err := s.Serve(listener)
		klog.Fatalf("failed to listen: %v", err)
	}()
	healthz := &server.HealthZ{
		Service: kmsServer,
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", strconv.FormatUint(uint64(*healthzPort), 10)),
			Path: *healthzPath,
		},
		UnixSocketPath: listener.Addr().String(),
		RPCTimeout:     *healthzTimeout,
	}
	go healthz.Serve()
	<-ctx.Done()
	// gracefully stop the grpc server
	klog.Infof("terminating the server")
	klog.Flush()
	// using os.Exit skips running deferred functions
	os.Exit(0)

}

// withShutdownSignal returns a copy of the parent context that will close if
// the process receives termination signals.
func withShutdownSignal(ctx context.Context) context.Context {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	nctx, cancel := context.WithCancel(ctx)

	go func() {
		<-signalChan
		klog.Info("received shutdown signal")
		cancel()
	}()
	return nctx
}
