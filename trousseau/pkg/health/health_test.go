package health

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/ondat/trousseau/pkg/providers"
	"github.com/ondat/trousseau/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	pb "k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1"
)

func TestHealthz(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.Nil(t, err)

	defer os.Remove(tmpDir)

	socket := path.Join(tmpDir, "healthz.socket")

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		s := grpc.NewServer()
		pb.RegisterKeyManagementServiceServer(s, &providers.KeyManagementServiceServer{
			Client: &mockClient{},
		})

		listener, listenErr := net.Listen("unix", socket)
		require.Nil(t, listenErr)

		wg.Done()

		err = s.Serve(listener)
		require.Nil(t, err)
	}()

	wg.Wait()

	req, err := http.NewRequest("GET", "", http.NoBody)
	require.Nil(t, err)

	writer := httptest.NewRecorder()

	service := Service{
		Service: &mockProvider{},
		HealthCheckURL: &url.URL{
			Host: net.JoinHostPort("", "8787"),
			Path: "/healthz",
		},
		UnixSocketPath: socket,
		Timeout:        time.Second,
	}
	service.ServeHTTP(writer, req)

	content, err := io.ReadAll(writer.Body)

	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, writer.Code)
	assert.Equal(t, "ok", string(content))
}

type mockProvider struct{}

func (m *mockProvider) Decrypt(_ context.Context, dr *pb.DecryptRequest) (*pb.DecryptResponse, error) {
	return &pb.DecryptResponse{
		Plain: dr.Cipher,
	}, nil
}

func (m *mockProvider) Encrypt(_ context.Context, er *pb.EncryptRequest) (*pb.EncryptResponse, error) {
	return &pb.EncryptResponse{
		Cipher: er.Plain,
	}, nil
}

func (m *mockProvider) Version(context.Context, *pb.VersionRequest) (*pb.VersionResponse, error) {
	return &pb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.BuildVersion,
	}, nil
}

type mockClient struct{}

func (s *mockClient) Encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (s *mockClient) Decrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (s *mockClient) Version() *pb.VersionResponse {
	return &pb.VersionResponse{
		Version:        version.APIVersion,
		RuntimeName:    version.Runtime,
		RuntimeVersion: version.BuildVersion,
	}
}
