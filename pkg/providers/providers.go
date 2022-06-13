package providers

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ondat/trousseau/pkg/logger"
	"k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

// EncryptionClient is the main interface for provider client.
type EncryptionClient interface {
	Decrypt(data []byte) ([]byte, error)
	Encrypt(data []byte) ([]byte, error)
	Version() *v1beta1.VersionResponse
}

// KeyManagementService is the main interface for gRPC server.
type KeyManagementService interface {
	Decrypt(context.Context, *v1beta1.DecryptRequest) (*v1beta1.DecryptResponse, error)
	Encrypt(context.Context, *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error)
	Version(context.Context, *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error)
}

// KeyManagementServiceServer base implementation if gRPC server.
type KeyManagementServiceServer struct {
	Client EncryptionClient
}

// Encrypt encryption requet handler.
func (k *KeyManagementServiceServer) Encrypt(ctx context.Context, data *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error) {
	klog.V(logger.Debug1).Info("Encrypt has been called...")

	plain := base64.StdEncoding.EncodeToString(data.Plain)

	response, err := k.Client.Encrypt([]byte(plain))
	if err != nil {
		klog.InfoS("Failed to encrypt", "error", err.Error())
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	klog.V(logger.Debug1).Info("Encrypt request complete")

	return &v1beta1.EncryptResponse{Cipher: response}, nil
}

// Decrypt decryption requet handler.
func (k *KeyManagementServiceServer) Decrypt(ctx context.Context, data *v1beta1.DecryptRequest) (*v1beta1.DecryptResponse, error) {
	klog.V(logger.Debug1).Info("Decrypt has been called...")

	klog.V(logger.Debug1).Info("Decrypt request starting...")

	r, err := k.Client.Decrypt(data.Cipher)
	if err != nil {
		klog.InfoS("Failed to decrypt", "error", err.Error())
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	w, err := base64.StdEncoding.DecodeString(string(r))
	if err != nil {
		klog.InfoS("Failed decode encrypted data", "error", err.Error())
		return nil, fmt.Errorf("failed decode encrypted data: %w", err)
	}

	klog.V(logger.Debug1).Info("Decrypt request complete")

	return &v1beta1.DecryptResponse{Plain: w}, nil
}

// Version version of gRPS server.
func (k *KeyManagementServiceServer) Version(context.Context, *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error) {
	return k.Client.Version(), nil
}
