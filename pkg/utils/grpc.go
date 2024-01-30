package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

// UnaryServerInterceptor provides metrics around Unary RPCs.
func UnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		klog.InfoS("GRPC request error", "method", info.FullMethod, "error", err.Error())

		return nil, fmt.Errorf("GRPC request error: %w", err)
	}

	return resp, nil
}
