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
	"strings"
	"syscall"
	"time"

	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/metrics"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/pkg/version"
	"github.com/ondat/trousseau/trousseau/pkg/health"
	"github.com/ondat/trousseau/trousseau/pkg/server"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"

	"k8s.io/klog/v2"
)

const hostPortFormatBase = 10

const (
	maxAllowedProviders      = 2
	defaultHealthzTimeout    = 10 * time.Second
	defaultSocketTimeout     = 20 * time.Second
	defaultHealthPort        = 8787
	defaultMetricsPort       = "8095"
	defaultDecryptPreference = "roundrobin"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var enabledProviders arrayFlags

	flag.Var(&enabledProviders, "enabled-providers", "list of enabled providers")
	socketLocation := flag.String("socket-location", "/opt/trousseau-kms", "location of provider sockets")
	socketTimeout := flag.Duration("socket-timeout", defaultSocketTimeout, "RPC timeout for provider socket")
	listenAddr := flag.String("listen-addr", "unix:///opt/trousseau-kms/trousseau.socket", "gRPC listen address")
	logEncoder := flag.String("zap-encoder", "console", "set log encoder [console, json]")
	healthzPort := flag.Int("healthz-port", defaultHealthPort, "port for health check")
	healthzPath := flag.String("healthz-path", "/healthz", "path for health check")
	healthzTimeout := flag.Duration("healthz-timeout", defaultHealthzTimeout, "RPC timeout for health check")
	metricsBackend := flag.String("metrics-backend", "prometheus", "Backend used for metrics")
	metricsAddress := flag.String("metrics-addr", defaultMetricsPort, "The address the metric endpoint binds to")
	decryptPreference := flag.String("decrypt-preference", defaultDecryptPreference, "The decrypt provider preference [roundrobin, fastest]")

	flag.Parse()

	err := logger.InitializeLogging(*logEncoder)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	if len(enabledProviders) > maxAllowedProviders {
		klog.Errorln(fmt.Errorf("max allowed providers: %d", maxAllowedProviders))
		os.Exit(1)
	}

	ctx := withShutdownSignal(context.Background())

	// initialize metrics exporter
	go func() {
		//nolint:govet // We know err is a shadow
		err := metrics.Serve(*metricsBackend, *metricsAddress)
		if err != nil {
			klog.Errorln(err)
			os.Exit(1)
		}

		klog.Fatalln("metrics service has stopped gracefully")
	}()

	klog.InfoS("Starting VaultEncryptionServiceServer service", "version", version.BuildVersion, "buildDate", version.BuildDate)

	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	if err = utils.RemoveFile(addr); err != nil {
		klog.ErrorS(err, "unable to delete socket file", "file", addr)
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	klog.InfoS("Listening for connections", "address", listener.Addr())

	go func() {
		klog.Errorln(<-utils.WatchFile(addr))
		os.Exit(1)
	}()

	kmsServer, err := server.New(*decryptPreference, *socketLocation, enabledProviders, *socketTimeout)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, kmsServer)

	go func() {
		if err := s.Serve(listener); err != nil {
			klog.Errorln(err)
			os.Exit(1)
		}

		klog.Fatalln("GRPC service has stopped gracefully")
	}()

	healthz := &health.Service{
		Service: kmsServer,
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", strconv.FormatUint(uint64(*healthzPort), hostPortFormatBase)),
			Path: *healthzPath,
		},
		UnixSocketPath: listener.Addr().String(),
		Timeout:        *healthzTimeout,
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
