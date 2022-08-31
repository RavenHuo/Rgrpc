/**
 * @Author raven
 * @Description
 * @Date 2022/8/30
 **/
package options

import (
	"context"
	"google.golang.org/grpc"
	"time"
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

type CallOptions struct {
	callOptions []grpc.CallOption
	timeout     time.Duration
	ctx         context.Context
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

func (o *CallOptions) ParentContext() context.Context {
	return o.ctx
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
