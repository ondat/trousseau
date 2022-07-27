package main

import (
	"flag"
	"net"
	"os"

	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/providers/vault/pkg/vault"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

var (
	listenAddr     = flag.String("listen-addr", "unix:///opt/trousseau-kms/vault/vault.socket", "gRPC listen address")
	configFilePath = flag.String("config-file-path", "/opt/trousseau-kms/vault/config.yaml", "Path for Vault Provider config file")
	logEncoder     = flag.String("zap-encoder", "console", "set log encoder [console, json]")
)

func main() {
	flag.Parse()

	err := logger.InitializeLogging(*logEncoder)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	cfg := vault.Config{}
	if err = utils.ParseConfig(*configFilePath, &cfg); err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	client, err := vault.New(&cfg)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, &providers.KeyManagementServiceServer{
		Client: client,
	})

	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	if err = os.Remove(addr); err != nil && !os.IsNotExist(err) {
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

	if err := s.Serve(listener); err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	klog.Fatalln("GRPC service has stopped gracefully")
}
