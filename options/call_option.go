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
	"time"
)

const (
	defaultTimeout = 10
)

type CallOption func(*CallOptions)

func WithGRPCCallOption(calls ...grpc.CallOption) CallOption {
	return func(o *CallOptions) {
		o.callOptions = append(o.callOptions, calls...)
	}
}

func WithTimeout(timeout time.Duration) CallOption {
	return func(o *CallOptions) {
		o.timeout = timeout
	}
}

func WithContext(ctx context.Context) CallOption {
	return func(o *CallOptions) {
		o.ctx = ctx
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

func DefaultCallOptions() *CallOptions {
	opts := &CallOptions{}
	opts.timeout = defaultTimeout
	opts.callOptions = make([]grpc.CallOption, 0)
	interceptors := make([]grpc.UnaryClientInterceptor, 0)
	opts.interceptors = interceptors
	opts.retryTimes = 3
	opts.retryCode = []codes.Code{codes.Unavailable}
	opts.keepalive = 1
	opts.balanceName = roundrobin.Name
	return opts
}

type CallOptions struct {
	callOptions  []grpc.CallOption
	timeout      time.Duration
	ctx          context.Context
	interceptors []grpc.UnaryClientInterceptor
	conn         *grpc.ClientConn
	serviceName  string
	retryTimes   int32
	retryCode    []codes.Code
	keepalive    int32
	balanceName  string
}

func (o *CallOptions) Timeout() time.Duration {
	return o.timeout
}

func (o *CallOptions) GrpcCallOption() []grpc.CallOption {
	return o.callOptions
}

func (o *CallOptions) Context() (context.Context, context.CancelFunc) {
	var ctx context.Context
	var cancel context.CancelFunc
	if o.ParentContext() != nil {
		ctx, cancel = context.WithTimeout(o.ParentContext(), o.Timeout())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), o.Timeout())
	}
	return ctx, cancel
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

func (o *CallOptions) ParentContext() context.Context {
	return o.ctx
}

func (o *CallOptions) Keepalive() int32 {
	return o.keepalive
}

func (o *CallOptions) BalanceName() string {
	return o.balanceName
}

func CallOptionsWith(options ...CallOption) *CallOptions {
	ops := &CallOptions{
		callOptions: make([]grpc.CallOption, 0),
		timeout:     time.Second * 10,
	}
	for _, o := range options {
		o(ops)
	}
	return ops
}
