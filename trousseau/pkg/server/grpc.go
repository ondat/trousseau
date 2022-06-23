package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ondat/trousseau/pkg/logger"
	"github.com/ondat/trousseau/pkg/metrics"
	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/utils"
	"github.com/ondat/trousseau/pkg/version"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
	"k8s.io/klog/v2"
)

const separator = ":-:"

type registeredProviders map[string]func(*pb.EncryptRequest, *pb.DecryptRequest) ([]byte, error)

type providersService struct {
	providers          registeredProviders
	sortProviders      func() []string
	fastestMetricsChan chan<- Metric
	metricsReporter    metrics.StatsReporter
}

// New creates an instance of the KMS Service Server.
func New(decryptPreference, socketLocation string, enabledProviders []string, timeout time.Duration) (providers.KeyManagementService, error) {
	klog.V(logger.Debug1).Info("Initialize new providers service")

	service := providersService{
		metricsReporter: metrics.NewStatsReporter(),
	}

	switch decryptPreference {
	case "roundrobin":
		service.sortProviders = NewRoundrobin(enabledProviders).Next
	case "fastest":
		fastest := NewFastest(enabledProviders)
		service.sortProviders = fastest.Fastest
		service.fastestMetricsChan = fastest.C()
	default:
		return nil, fmt.Errorf("selected decryption preference isn't supported: %s", decryptPreference)
	}

	registered := registeredProviders{}

	for _, provider := range enabledProviders {
		provider := provider

		socket := filepath.Clean(filepath.Join(socketLocation, provider, fmt.Sprintf("%s.socket", provider)))
		if _, err := os.Stat(socket); err != nil {
			klog.ErrorS(err, "Unable to find socket", "name", provider, "socket", socket, "error", err.Error())
			return nil, fmt.Errorf("unable to find socket at %s: %w", socket, err)
		}

		registered[provider] = func(encReq *pb.EncryptRequest, decReq *pb.DecryptRequest) ([]byte, error) {
			klog.V(logger.Debug1).InfoS("Calling provider", "name", provider, "socket", socket, "encryption", encReq != nil, "decryption", decReq != nil)

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			conn, err := utils.DialUnixSocket(socket)
			if err != nil {
				klog.ErrorS(err, "Failed to call unix socket", "name", provider, "socket", socket, "encryption", encReq != nil, "decryption", decReq != nil)
				return nil, err
			}
			defer conn.Close()

			kmsClient := pb.NewKeyManagementServiceClient(conn)

			switch {
			case encReq != nil:
				res, err := kmsClient.Encrypt(ctx, encReq)
				if err != nil {
					klog.InfoS("Unable to encrypt data", "name", provider, "socket", socket)
					return nil, err
				}

				return res.Cipher, err
			case decReq != nil:
				res, err := kmsClient.Decrypt(ctx, decReq)
				if err != nil {
					klog.InfoS("Unable to decrypt data", "name", provider, "socket", socket)
					return nil, err
				}

				return res.Plain, err
			default:
				return nil, nil
			}
		}
	}

	service.providers = registered

	return &service, nil
}

// Encrypt encryption requet handler.
func (k *providersService) Encrypt(ctx context.Context, data *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	klog.V(logger.Debug1).Info("Encrypt has been called...")

	encrypted := map[string][]byte{}

	for name := range k.providers {
		klog.V(logger.Info3).InfoS("Encrypting...", "name", name)

		start := time.Now()

		provider := k.providers[name]

		r, err := provider(&pb.EncryptRequest{
			Version: data.Version,
			Plain:   data.Plain,
		}, nil)
		if err != nil {
			klog.InfoS("Failed to encrypt", "name", name, "error", err.Error())
			k.metricsReporter.ReportRequest(ctx, name, metrics.EncryptOperationTypeValue, metrics.ErrorStatusTypeValue, time.Since(start).Seconds(), err.Error())

			return nil, fmt.Errorf("failed to encrypt %s: %w", name, err)
		}

		k.metricsReporter.ReportRequest(ctx, name, metrics.EncryptOperationTypeValue, metrics.SuccessStatusTypeValue, time.Since(start).Seconds())

		encrypted[name] = r
	}

	final := strings.Builder{}
	for name, enc := range encrypted {
		if _, err := final.WriteString(fmt.Sprintf("%s%s%s\n", name, separator, string(enc))); err != nil {
			klog.InfoS("Failed to append result", "name", name, "error", err.Error())

			return nil, fmt.Errorf("failed to append result %s: %w", name, err)
		}
	}

	klog.V(logger.Debug1).Info("Encrypt request complete")

	return &pb.EncryptResponse{Cipher: []byte(final.String())}, nil
}

// Decrypt decryption requet handler.
func (k *providersService) Decrypt(ctx context.Context, data *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	klog.V(logger.Debug1).Info("Decrypt has been called...")

	const nParts = 2

	secrets := map[string]string{}

	for _, line := range strings.Split(string(data.Cipher), "\n") {
		parts := strings.Split(line, separator)
		if len(parts) != nParts {
			klog.InfoS("Failed to find proper decryption")
			continue
		}

		secrets[parts[0]] = parts[1]
	}

	for _, name := range k.sortProviders() {
		secret, ok := secrets[name]
		if !ok {
			klog.InfoS("Failed to find encrypted for provider", "name", name)

			continue
		}

		klog.V(logger.Info3).InfoS("Decrypting...", "name", name)

		start := time.Now()

		provider, ok := k.providers[name]
		if !ok {
			klog.InfoS("Failed to find provider", "name", name)

			continue
		}

		response, err := provider(nil, &pb.DecryptRequest{
			Version: data.Version,
			Cipher:  []byte(secret),
		})
		if err != nil {
			klog.InfoS("Failed to decrypt", "name", name, "error", err.Error())
			k.metricsReporter.ReportRequest(ctx, name, metrics.EncryptOperationTypeValue, metrics.ErrorStatusTypeValue, time.Since(start).Seconds(), err.Error())

			if k.fastestMetricsChan != nil {
				k.fastestMetricsChan <- Metric{
					Provider:    name,
					ReponseTime: time.Minute,
				}
			}

			continue
		}

		k.metricsReporter.ReportRequest(ctx, name, metrics.EncryptOperationTypeValue, metrics.SuccessStatusTypeValue, time.Since(start).Seconds())

		if k.fastestMetricsChan != nil {
			k.fastestMetricsChan <- Metric{
				Provider:    name,
				ReponseTime: time.Since(start),
			}
		}

		klog.V(logger.Debug1).Info("Decrypt request complete")

		return &pb.DecryptResponse{Plain: response}, nil
	}

	klog.InfoS("Failed to decrypt with all providers")

	return nil, errors.New("failed to decrypt with all providers")
}

// Version version of gRPS server.
func (k *providersService) Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{Version: version.APIVersion, RuntimeName: version.Runtime, RuntimeVersion: version.BuildVersion}, nil
}
