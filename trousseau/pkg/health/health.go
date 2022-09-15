package health

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"errors"

	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/pkg/version"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const (
	healthCheckPlainText = "healthcheck"
)

// Service Healtz endpoint.
type Service struct {
	Service        providers.KeyManagementService
	HealthCheckURL *url.URL
	UnixSocketPath string
	Timeout        time.Duration
}

// Serve creates the http handler for serving health requests.
func (h *Service) Serve() error {
	klog.V(logger.Info3).Info("Initialize health check")

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(h.HealthCheckURL.EscapedPath(), h.ServeHTTP)

	const timeout = time.Second * 5

	srv := &http.Server{
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout,
		Addr:              h.HealthCheckURL.Host,
		Handler:           serveMux,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		klog.Error(err, "Failed to start health check")
		return fmt.Errorf("failed to start health check server: %w", err)
	}

	return nil
}

// ServeHTTP is handling healthz requests.
func (h *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	klog.V(logger.Debug1).Info("Started health check...")

	ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
	defer cancel()

	conn, err := utils.DialUnixSocket(h.UnixSocketPath)
	if err != nil {
		klog.Error(err, "Failed to call unix socket")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)

		return
	}
	defer conn.Close()

	kmsClient := pb.NewKeyManagementServiceClient(conn)

	if err = h.checkRPC(ctx, kmsClient); err != nil {
		klog.Error(err, "Failed to check RPC")
		http.Error(w, err.Error(), http.StatusServiceUnavailable)

		return
	}

	enc, err := h.Service.Encrypt(ctx, &pb.EncryptRequest{Plain: []byte(healthCheckPlainText)})
	if err != nil {
		klog.Error(err, "Failed to encrypt", "data", healthCheckPlainText)
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	dec, err := h.Service.Decrypt(ctx, &pb.DecryptRequest{Cipher: enc.Cipher})
	if err != nil {
		klog.Error(err, "Failed to decrypt encrypted data")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	} else if string(dec.Plain) != healthCheckPlainText {
		klog.ErrorS(errors.New("failed to properly decrypt encrypted data"), "Encryption failed", "original", healthCheckPlainText, "decrypted", string(dec.Plain))
		http.Error(w, "plain text mismatch after decryption", http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	klog.V(logger.Debug1).Info("Completed health check")
}

// checkRPC initiates a grpc request to validate the socket is responding
// sends a KMS VersionRequest and checks if the VersionResponse is valid.
func (h *Service) checkRPC(ctx context.Context, client pb.KeyManagementServiceClient) error {
	v, err := client.Version(ctx, &pb.VersionRequest{})
	if err != nil {
		return fmt.Errorf("unable to get version: %w", err)
	}

	if v.Version != version.APIVersion || v.RuntimeName != version.Runtime || v.RuntimeVersion != version.BuildVersion {
		return errors.New("failed to get correct version response")
	}

	return nil
}
