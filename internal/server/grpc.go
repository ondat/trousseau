package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/ondat/trousseau/internal/config"
	"github.com/ondat/trousseau/internal/encrypt"
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
	klog.V(2).Infof("decrypt request started ")
	r, err := k.kvClient.Decrypt(data.Cipher)
	if err != nil {
		klog.ErrorS(err, "failed to decrypt")
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	w, err := base64.StdEncoding.DecodeString(string(r))
	if err != nil {
		klog.ErrorS(err, "failed decode encrypted data")
		return nil, fmt.Errorf("failed decode encrypted data: %w", err)
	}
	klog.V(2).Infof("decrypt request complete")
	return &v1beta1.DecryptResponse{Plain: w}, nil
}

func (k *keyManagementServiceServer) Encrypt(ctx context.Context, data *v1beta1.EncryptRequest) (*v1beta1.EncryptResponse, error) {
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
	klog.V(2).Infof("encrypt request started")
	plain := base64.StdEncoding.EncodeToString(data.Plain)
	response, err := k.kvClient.Encrypt([]byte(plain))
	if err != nil {
		klog.ErrorS(err, "failed to encrypt")
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}
	klog.V(2).Infof("encrypt request complete")
	return &v1beta1.EncryptResponse{Cipher: response}, nil
}

func (k *keyManagementServiceServer) Version(context.Context, *v1beta1.VersionRequest) (*v1beta1.VersionResponse, error) {
	return &v1beta1.VersionResponse{Version: version.APIVersion, RuntimeName: version.Runtime, RuntimeVersion: version.BuildVersion}, nil
}
