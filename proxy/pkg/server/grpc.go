package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/pkg/version"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

type proxyService struct {
	trousseauSocket string
	timeout         time.Duration
}

// New creates an instance of the Proxy Server.
func New(trousseauSocket string, timeout time.Duration) providers.KeyManagementService {
	return &proxyService{
		trousseauSocket: trousseauSocket,
		timeout:         timeout,
	}
}

// Encrypt encryption requet handler.
func (k *proxyService) Encrypt(ctx context.Context, data *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	conn, err := utils.DialUnixSocket(k.trousseauSocket)
	if err != nil {
		log.Printf("Unable to connect to %s: %s", k.trousseauSocket, err.Error())

		return nil, err
	}
	defer conn.Close()

	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), k.timeout)

		kmsClient := pb.NewKeyManagementServiceClient(conn)

		res, err := kmsClient.Encrypt(ctx, data)

		cancel()

		if err != nil {
			log.Printf("Unable to encrypt data: %s", err.Error())

			continue
		}

		return res, err
	}

	return nil, fmt.Errorf("connection error on %s", k.trousseauSocket)
}

// Decrypt decryption requet handler.
func (k *proxyService) Decrypt(ctx context.Context, data *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	conn, err := utils.DialUnixSocket(k.trousseauSocket)
	if err != nil {
		log.Printf("Unable to connect to %s: %s", k.trousseauSocket, err.Error())

		return nil, err
	}

	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), k.timeout)

		kmsClient := pb.NewKeyManagementServiceClient(conn)

		res, err := kmsClient.Decrypt(ctx, data)

		cancel()

		if err != nil {
			log.Printf("Unable to decrypt data: %s", err.Error())

			continue
		}

		return res, err
	}

	return nil, fmt.Errorf("connection error on %s", k.trousseauSocket)
}

// Version version of gRPS server.
func (k *proxyService) Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{Version: version.APIVersion, RuntimeName: version.Runtime, RuntimeVersion: version.BuildVersion}, nil
}
