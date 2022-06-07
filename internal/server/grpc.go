package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/ondat/trousseau/internal/config"
	"github.com/ondat/trousseau/internal/encrypt"
	"github.com/ondat/trousseau/internal/logger"
	"github.com/ondat/trousseau/internal/metrics"
	"github.com/ondat/trousseau/internal/version"
	"k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

type KeyManagementService interface {
	Decrypt(context.Context, *v1beta1.DecryptRequest) (*v1beta1.DecryptResponse, error)
	Encrypt(context.Context, *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error)
	Version(context.Context, *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error)
}
type keyManagementServiceServer struct {
	kvClient encrypt.EncryptionClient
	reporter metrics.StatsReporter
}

// New creates an instance of the KMS Service Server.
func New(ctx context.Context, cfg config.ProviderConfig) (KeyManagementService, error) {
	klog.V(logger.Debug1).Info("Initialize new GRPC service")

	kvClient, err := encrypt.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create encrypt service: %w", err)
	}

	return &keyManagementServiceServer{
		kvClient: kvClient,
		reporter: metrics.NewStatsReporter(),
	}, nil
}

func (k *keyManagementServiceServer) Decrypt(ctx context.Context, data *v1beta1.DecryptRequest) (*v1beta1.DecryptResponse, error) {
	klog.V(logger.Debug1).Info("Decrypt has been called...")

	start := time.Now()

	var err error
	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue

		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}

		k.reporter.ReportRequest(ctx, metrics.DecryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
	}()
	klog.V(logger.Debug1).Info("Decrypt request starting...")

	r, err := k.kvClient.Decrypt(data.Cipher)
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

func (k *keyManagementServiceServer) Encrypt(ctx context.Context, data *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error) {
	klog.V(logger.Debug1).Info("Encrypt has been called...")

	start := time.Now()

	var err error
	defer func() {
		errors := ""
		status := metrics.SuccessStatusTypeValue

		if err != nil {
			status = metrics.ErrorStatusTypeValue
			errors = err.Error()
		}

		k.reporter.ReportRequest(ctx, metrics.EncryptOperationTypeValue, status, time.Since(start).Seconds(), errors)
	}()
	klog.V(logger.Debug1).Info("Encrypt request starting...")

	plain := base64.StdEncoding.EncodeToString(data.Plain)

	response, err := k.kvClient.Encrypt([]byte(plain))
	if err != nil {
		klog.InfoS("Failed to encrypt", "error", err.Error())
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	klog.V(logger.Debug1).Info("Encrypt request complete")

	return &v1beta1.EncryptResponse{Cipher: response}, nil
}

func (k *keyManagementServiceServer) Version(context.Context, *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error) {
	return &v1beta1.VersionResponse{Version: version.APIVersion, RuntimeName: version.Runtime, RuntimeVersion: version.BuildVersion}, nil
}
