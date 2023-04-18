/**
 * @Author raven
 * @Description
 * @Date 2023/4/18
 **/
package options

import "google.golang.org/grpc"

type GrpcOptions struct {
	serverOptions      []grpc.ServerOption
	streamInterceptors []grpc.StreamServerInterceptor
	unaryInterceptors  []grpc.UnaryServerInterceptor
}

type GrpcOption func(options *GrpcOptions)

func (o *GrpcOptions) ServerOptions() []grpc.ServerOption {
	return o.serverOptions
}
func (o *GrpcOptions) StreamInterceptors() []grpc.StreamServerInterceptor {
	return o.streamInterceptors
}
func (o *GrpcOptions) UnaryInterceptors() []grpc.UnaryServerInterceptor {
	return o.unaryInterceptors
}
