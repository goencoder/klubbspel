package validate

import (
	"context"

	"buf.build/go/protovalidate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// ValidationInterceptor validates incoming gRPC requests using protovalidate.
func ValidationInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// If the request implements proto.Message, validate it.
	if msg, ok := req.(proto.Message); ok {
		if err := protovalidate.Validate(msg); err != nil {
			// Return an InvalidArgument error if validation fails.
			return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
		}
	}
	// Proceed to handle the request if validation passes.
	return handler(ctx, req)
}
