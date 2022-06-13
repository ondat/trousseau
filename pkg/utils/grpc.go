package utils

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/ondat/trousseau/pkg/logger"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	splitin = 2
)

// ParseEndpoint returns unix socket's protocol and address
func ParseEndpoint(ep string) (proto, address string, err error) {
	err = fmt.Errorf("invalid endpoint: %s", ep)

	if strings.HasPrefix(strings.ToLower(ep), "unix://") {
		s := strings.SplitN(ep, "://", splitin)
		if s[1] != "" {
			proto = s[0]
			address = s[1]
			err = nil
		}
	}

	return
}

// UnaryServerInterceptor provides metrics around Unary RPCs.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(logger.Debug1).InfoS("GRPC call", "method", info.FullMethod)

	resp, err := handler(ctx, req)
	if err != nil {
		klog.InfoS("GRPC request error", "method", info.FullMethod, "error", err.Error())

		return nil, fmt.Errorf("GRPC request error: %w", err)
	}

	return resp, nil
}

// DialUnixSocket creates a gRPC connection.
func DialUnixSocket(unixSocketPath string) (*grpc.ClientConn, error) {
	return grpc.Dial(
		unixSocketPath,
		//nolint:staticcheck // we know WithInsecure is deprecated
		grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, target string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", target)
		}),
	)
}
