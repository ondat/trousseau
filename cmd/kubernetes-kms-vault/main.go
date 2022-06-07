package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ondat/trousseau/internal/config"
	"github.com/ondat/trousseau/internal/logger"
	"github.com/ondat/trousseau/internal/metrics"
	"github.com/ondat/trousseau/internal/server"
	"github.com/ondat/trousseau/internal/utils"
	"github.com/ondat/trousseau/internal/version"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"

	// json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
)

const (
	defaultHealthzTimeout = 20 * time.Second
	hostPortFormatBase    = 10

	healthPort  = 8787
	metricsPort = "8095"
)

var (
	listenAddr     = flag.String("listen-addr", "unix:///opt/vaultkms.socket", "gRPC listen address")
	logEncoder     = flag.String("zap-encoder", "console", "set log encoder [console, json]")
	configFilePath = flag.String("config-file-path", "./config.yaml", "Path for Vault Provider config file")
	healthzPort    = flag.Int("healthz-port", healthPort, "port for health check")
	healthzPath    = flag.String("healthz-path", "/healthz", "path for health check")
	healthzTimeout = flag.Duration("healthz-timeout", defaultHealthzTimeout, "RPC timeout for health check")
	metricsBackend = flag.String("metrics-backend", "prometheus", "Backend used for metrics")
	metricsAddress = flag.String("metrics-addr", metricsPort, "The address the metric endpoint binds to")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	v := flag.CommandLine.Lookup("v").Value.String()

	logLevel, err := strconv.Atoi(v)
	if err != nil {
		klog.Fatalln("Invalid verbosity level", "level", v)
	}

	klog.SetLogger(logger.NewLogger(klog.Level(logLevel), *logEncoder))

	ctx := withShutdownSignal(context.Background())

	// initialize metrics exporter
	go func() {
		err := metrics.Serve(*metricsBackend, *metricsAddress)
		if err != nil {
			klog.Errorln(err)
			os.Exit(1)
		}

		klog.Fatalln("metrics service has stopped gracefully")
	}()

	klog.InfoS("Starting VaultEncryptionServiceServer service", "version", version.BuildVersion, "buildDate", version.BuildDate)

	cfg, err := config.New(*configFilePath)
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
		klog.Errorln(err)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	kmsServer, err := server.New(ctx, cfg)
	pb.RegisterKeyManagementServiceServer(s, kmsServer)

	if err != nil {
		klog.Errorln(fmt.Errorf("failed to listen: %w", err))
		os.Exit(1)
	}

	klog.InfoS("Listening for connections", "address", listener.Addr())

	go func() {
		if err := s.Serve(listener); err != nil {
			klog.Errorln(err)
			os.Exit(1)
		}

		klog.Fatalln("GRPC service has stopped gracefully")
	}()

	healthz := &server.HealthZ{
		Service: kmsServer,
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", strconv.FormatUint(uint64(*healthzPort), hostPortFormatBase)),
			Path: *healthzPath,
		},
		UnixSocketPath: listener.Addr().String(),
		RPCTimeout:     *healthzTimeout,
	}

	go func() {
		if err := healthz.Serve(); err != nil {
			klog.Errorln(err)
			os.Exit(1)
		}

		klog.Fatalln("healtz service has stopped gracefully")
	}()

	<-ctx.Done()
	// gracefully stop the grpc server
	klog.Info("Terminating the server")
	s.GracefulStop()
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
		klog.Info("Received shutdown signal")
		cancel()
	}()

	return nctx
}
