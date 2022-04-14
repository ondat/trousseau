package utils

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	klogv   = 2
	splitin = 2
)

// ParseEndpoint returns unix socket's protocol and address
func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") {
		s := strings.SplitN(ep, "://", splitin)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}

	return "", "", fmt.Errorf("invalid endpoint: %s", ep)
}

// UnaryServerInterceptor provides metrics around Unary RPCs.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(klogv).Infof("GRPC call: %s", info.FullMethod)
	resp, err := handler(ctx, req)

	if err != nil {
		klog.ErrorS(err, "GRPC request error")
		err = fmt.Errorf("GRPC request error: %w", err)
	}

	return resp, err
}

// func getGRPCMethodName(fullMethodName string) string {
// 	fullMethodName = strings.TrimPrefix(fullMethodName, "/")
// 	methodNames := strings.Split(fullMethodName, "/")
// 	if len(methodNames) >= 2 {
// 		return strings.ToLower(methodNames[1])
// 	}

// 	return "unknown"
// }
