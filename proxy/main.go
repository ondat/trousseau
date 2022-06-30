package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/proxy/pkg/server"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

const (
	defaultSocketTimeout = 5 * time.Second
)

var (
	listenAddr    = flag.String("listen-addr", "unix:///opt/vault-kms/proxy.socket", "gRPC listen address")
	trousseauAddr = flag.String("trousseau-addr", "/opt/vault-kms/trousseau.socket", "gRPC listen address")
	socketTimeout = flag.Duration("socket-timeout", defaultSocketTimeout, "RPC timeout for Trousseau socket")
)

func main() {
	flag.Parse()

	proto, addr, err := utils.ParseEndpoint(*listenAddr)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err = utils.RemoveFile(addr); err != nil {
		log.Printf("Unable to delete socket file: %s: %s\n", addr, err.Error())
	}

	listener, err := net.Listen(proto, addr)
	if err != nil {
		log.Fatal(err.Error())
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(utils.UnaryServerInterceptor),
	}
	s := grpc.NewServer(opts...)
	pb.RegisterKeyManagementServiceServer(s, server.New(*trousseauAddr, *socketTimeout))

	log.Println("Listening for connections on address: " + listener.Addr().String())

	if err := s.Serve(listener); err != nil {
		log.Fatal(err.Error())
	}

	log.Println("GRPC service has stopped gracefully")
}
