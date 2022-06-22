package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const logEncoder = "console"

var (
	listenAddr = flag.String("listen-addr", "unix:///opt/vault-kms/debug/debug.socket", "gRPC listen address")
)

func main() {
	flag.Parse()

	err := logger.InitializeLogging(logEncoder)
	if err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}

	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, &providers.KeyManagementServiceServer{
		Client: &service{},
	})

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

	klog.InfoS("Listening for connections", "address", listener.Addr())

	if err := s.Serve(listener); err != nil {
		klog.Errorln(err)
		os.Exit(1)
	}

	klog.Fatalln("GRPC service has stopped gracefully")
}

type service struct{}

func (s *service) Encrypt(data []byte) ([]byte, error) {
	klog.InfoS("Encrypt", "data", string(data))

	return []byte(base64.StdEncoding.EncodeToString(data)), nil
}

func (s *service) Decrypt(data []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		klog.InfoS("Failed decode encrypted data", "error", err.Error())
		return nil, fmt.Errorf("failed decode encrypted data: %w", err)
	}

	klog.InfoS("Decrypt", "data", string(decoded))

	return decoded, nil
}

func (s *service) Version() *pb.VersionResponse {
	return &pb.VersionResponse{
		Version:        "debug",
		RuntimeName:    "debug",
		RuntimeVersion: "0.0.0",
	}
}
