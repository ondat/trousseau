package utils

import (
	"context"
	"fmt"
	"strings"

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
	klog.V(4).InfoS("GRPC call", "method", info.FullMethod)

	resp, err := handler(ctx, req)
	if err != nil {
		klog.InfoS("GRPC request error", "method", info.FullMethod, "error", err.Error())
		return nil, fmt.Errorf("GRPC request error: %w", err)
	}

	return resp, nil
}
