/**
 * @Author raven
 * @Description
 * @Date 2022/8/30
 **/
package options

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"time"
)

const (
	defaultTimeout = 10
)

type CallOption func(*CallOptions)

func WithTimeout(timeout time.Duration) CallOption {
	return func(o *CallOptions) {
		o.timeout = timeout
	}
}

func WithIntercept(interceptors []grpc.UnaryClientInterceptor) CallOption {
	return func(o *CallOptions) {
		o.interceptors = append(o.interceptors, interceptors...)
	}
}

func WithServiceName(serviceName string) CallOption {
	return func(options *CallOptions) {
		options.serviceName = serviceName
	}
}

func WithDialTimeout(dialTimeout time.Duration) CallOption {
	return func(options *CallOptions) {
		options.dialTimeout = dialTimeout
	}
}

func WithCompressor(compressor string) CallOption {
	return func(options *CallOptions) {
		options.compressor = compressor
	}
}

func DefaultCallOptions() *CallOptions {
	opts := &CallOptions{}
	opts.timeout = defaultTimeout * time.Second
	opts.dialTimeout = defaultTimeout * time.Second
	interceptors := make([]grpc.UnaryClientInterceptor, 0)
	opts.interceptors = interceptors
	opts.retryTimes = 3
	opts.retryCode = []codes.Code{codes.Unavailable}
	opts.keepalive = 1
	opts.balanceName = roundrobin.Name
	opts.compressor = gzip.Name
	return opts
}

type CallOptions struct {
	// 调用超时时间
	timeout time.Duration
	// grpc链接的超时时间
	dialTimeout  time.Duration
	interceptors []grpc.UnaryClientInterceptor
	// 需要调用的服务名
	serviceName string
	// 重试次数
	retryTimes int32
	// 重试的code
	retryCode []codes.Code
	// grpc keepalive的时间
	keepalive int32
	// 负载均衡策略
	balanceName string
	// 压缩传输算法
	compressor string
}

func (o *CallOptions) Timeout() time.Duration {
	return o.timeout
}
func (o *CallOptions) DialTimeout() time.Duration {
	return o.dialTimeout
}

func (o *CallOptions) GrpcCallOption() []grpc.CallOption {
	grpcCallOption := make([]grpc.CallOption, 0)
	grpcCallOption = append(grpcCallOption, grpc.UseCompressor(o.compressor))
	return grpcCallOption
}

func (o *CallOptions) Interceptors() []grpc.UnaryClientInterceptor {
	return o.interceptors
}

func (o *CallOptions) ServiceName() string {
	return o.serviceName
}
func (o *CallOptions) RetryTimes() int32 {
	return o.retryTimes
}
func (o *CallOptions) RetryCodes() []codes.Code {
	return o.retryCode
}

func (o *CallOptions) Keepalive() int32 {
	return o.keepalive
}

func (o *CallOptions) BalanceName() string {
	return o.balanceName
}

func (o *CallOptions) Compressor() string {
	return o.compressor
}

func GrpcCallOption(opts ...grpc.CallOption) []grpc.CallOption {
	ops := DefaultCallOptions().GrpcCallOption()
	ops = append(ops, opts...)
	return opts
}

func GrpcTimeoutCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := DefaultCallOptions().Timeout()
	return context.WithTimeout(ctx, timeout)
}
