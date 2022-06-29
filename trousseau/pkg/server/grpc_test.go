package server

import (
	"context"
	"io/fs"
	"net"
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

func TestGRPCService(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.Nil(t, err)

	defer os.Remove(tmpDir)

	socketDir := path.Join(tmpDir, "debug")
	err = os.Mkdir(socketDir, fs.ModePerm)
	require.Nil(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		s := grpc.NewServer()
		pb.RegisterKeyManagementServiceServer(s, &providers.KeyManagementServiceServer{
			Client: &mockClient{},
		})

		listener, listenErr := net.Listen("unix", path.Join(socketDir, "debug.socket"))
		require.Nil(t, listenErr)

		wg.Done()

		err = s.Serve(listener)
		require.Nil(t, err)
	}()

	wg.Wait()

	service, err := New("roundrobin", tmpDir, []string{"debug"}, time.Second)
	require.Nil(t, err)

	decrypted, err := service.Encrypt(context.Background(), &pb.EncryptRequest{
		Plain: []byte("foobar"),
	})
	require.Nil(t, err)

	assert.Equal(t, []byte("debug:-:foobar\n"), decrypted.Cipher, "Invalid encryption")

	encrypted, err := service.Decrypt(context.Background(), &pb.DecryptRequest{
		Cipher: decrypted.Cipher,
	})
	require.Nil(t, err)

	assert.Equal(t, []byte("foobar"), encrypted.Plain, "Invalid decryption")
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
