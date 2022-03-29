package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	errors "errors"

	"github.com/ondat/trousseau/internal/version"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const (
	healthCheckPlainText = "healthcheck"
)

type HealthZ struct {
	Service        KeyManagementService
	HealthCheckURL *url.URL
	UnixSocketPath string
	RPCTimeout     time.Duration
}

// Serve creates the http handler for serving health requests
func (h *HealthZ) Serve() error {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc(h.HealthCheckURL.EscapedPath(), h.ServeHTTP)
	if err := http.ListenAndServe(h.HealthCheckURL.Host, serveMux); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start health check server: %w", err)
	}

	return nil
}

func (h *HealthZ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	klog.V(5).Infof("Started health check")
	ctx, cancel := context.WithTimeout(context.Background(), h.RPCTimeout)
	defer cancel()

	conn, err := h.dialUnixSocket()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer conn.Close()

	kmsClient := pb.NewKeyManagementServiceClient(conn)
	err = h.checkRPC(ctx, kmsClient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	enc, err := h.Service.Encrypt(ctx, &pb.EncryptRequest{Plain: []byte(healthCheckPlainText)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dec, err := h.Service.Decrypt(ctx, &pb.DecryptRequest{Cipher: enc.Cipher})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if string(dec.Plain) != healthCheckPlainText {
		http.Error(w, "plain text mismatch after decryption", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	klog.V(5).Infof("Completed health check")
}

// checkRPC initiates a grpc request to validate the socket is responding
// sends a KMS VersionRequest and checks if the VersionResponse is valid.
func (h *HealthZ) checkRPC(ctx context.Context, client pb.KeyManagementServiceClient) error {
	v, err := client.Version(ctx, &pb.VersionRequest{})
	if err != nil {
		return fmt.Errorf("unable to get version: %w", err)
	}
	if v.Version != version.APIVersion || v.RuntimeName != version.Runtime || v.RuntimeVersion != version.BuildVersion {
		return errors.New("failed to get correct version response")
	}
	return nil
}

func (h *HealthZ) dialUnixSocket() (*grpc.ClientConn, error) {
	return grpc.Dial(
		h.UnixSocketPath,
		grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", target)
		}),
	)
}
