package providers

import (
	"context"
	"fmt"

	"github.com/ondat/trousseau/pkg/logger"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

// EncryptionClient is the main interface for provider client.
type EncryptionClient interface {
	Decrypt(data []byte) ([]byte, error)
	Encrypt(data []byte) ([]byte, error)
	Version() *pb.VersionResponse
}

// KeyManagementService is the main interface for gRPC server.
type KeyManagementService interface {
	Decrypt(context.Context, *pb.DecryptRequest) (*pb.DecryptResponse, error)
	Encrypt(context.Context, *pb.EncryptRequest) (*pb.EncryptResponse, error)
	Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error)
}

// KeyManagementServiceServer base implementation if gRPC server.
type KeyManagementServiceServer struct {
	Client EncryptionClient
}

// Encrypt encryption requet handler.
func (k *KeyManagementServiceServer) Encrypt(ctx context.Context, data *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	klog.V(logger.Debug1).Info("Encrypt has been called...")

	response, err := k.Client.Encrypt(data.Plain)
	if err != nil {
		klog.InfoS("Failed to encrypt", "error", err.Error())
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	klog.V(logger.Debug1).Info("Encrypt request complete")

	return &pb.EncryptResponse{Cipher: response}, nil
}

// Decrypt decryption requet handler.
func (k *KeyManagementServiceServer) Decrypt(ctx context.Context, data *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	klog.V(logger.Debug1).Info("Decrypt has been called...")

	response, err := k.Client.Decrypt(data.Cipher)
	if err != nil {
		klog.InfoS("Failed to decrypt", "error", err.Error())
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	klog.V(logger.Debug1).Info("Decrypt request complete")

	return &pb.DecryptResponse{Plain: response}, nil
}

// Version version of gRPS server.
func (k *KeyManagementServiceServer) Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error) {
	return k.Client.Version(), nil
}
