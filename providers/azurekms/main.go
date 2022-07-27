package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/kubernetes-kms/pkg/plugin"
	"github.com/Azure/kubernetes-kms/pkg/utils"
	"github.com/Azure/kubernetes-kms/pkg/version"
	"github.com/ondat/trousseau/pkg/logger"
	trutils "github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/providers/azurekms/pkg/azurekms"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

var (
	listenAddr     = flag.String("listen-addr", "unix:///opt/trousseau-kms/azurekms/azurekms.socket", "gRPC listen address")
	configFilePath = flag.String("config-file-path", "/opt/trousseau-kms/azurekms/config.yaml", "Path for Azure KMS Provider config file")
	logEncoder     = flag.String("zap-encoder", "console", "set log encoder [console, json]")
)

func main() {
	flag.Parse()

	err := logger.InitializeLogging(*logEncoder)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	cfg := azurekms.Config{}
	if err = trutils.ParseConfig(*configFilePath, &cfg); err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	ctx := withShutdownSignal(context.Background())

	klog.InfoS("Starting KeyManagementServiceServer service", "version", version.BuildVersion, "buildDate", version.BuildDate)

	pc := &plugin.Config{
		KeyVaultName:   cfg.KeyVaultName,
		KeyName:        cfg.KeyName,
		KeyVersion:     cfg.KeyVersion,
		ManagedHSM:     cfg.ManagedHMS,
		ConfigFilePath: cfg.ConfigFilePath,
	}

	kmsServer, err := plugin.New(ctx, pc)
	if err != nil {
		klog.ErrorS(err, "failed to create server")
		os.Exit(1)
	}

	// Initialize and run the GRPC server
	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		klog.ErrorS(err, "failed to parse endpoint")
		os.Exit(1)
	}

	if err = os.Remove(addr); err != nil && !os.IsNotExist(err) {
		klog.ErrorS(err, "failed to remove socket file", "addr", addr)
		os.Exit(1)
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		klog.ErrorS(err, "failed to listen", "addr", addr, "proto", proto)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, kmsServer)

	klog.InfoS("Listening for connections", "addr", listener.Addr().String())
	//nolint:errcheck // original implementation in plugin
	go s.Serve(listener)

	<-ctx.Done()
	// gracefully stop the grpc server
	klog.Info("terminating the server")
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
		klog.Info("received shutdown signal")
		cancel()
	}()

	return nctx
}
