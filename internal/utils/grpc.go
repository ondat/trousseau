package utils

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

// ParseEndpoint returns unix socket's protocol and address
func ParseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}
	return "", "", errors.Errorf("invalid endpoint: %s", ep)
}

// UnaryServerInterceptor provides metrics around Unary RPCs.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(5).Infof("GRPC call: %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		klog.ErrorS(err, "GRPC request error")
		err = errors.WithMessage(err, "GRPC request error")
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
