package Rgrpc

import (
	"context"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/intercept"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/go-kit/log"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"time"
)

func NewResolveConn(info *instance.ServerInfo, opts []options.CallOption) *grpc.ClientConn {
	option := options.DefaultCallOptions()
	for _, o := range opts {
		o(option)
	}
	interceptors := option.Interceptors()
	interceptors = append(interceptors, grpc_retry.UnaryClientInterceptor([]grpc_retry.CallOption{
		grpc_retry.WithBackoff(grpc_retry.BackoffExponential(10 * time.Millisecond)), // 指数退避
		grpc_retry.WithCodes(option.RetryCodes()...),                                 // 503才重试
		grpc_retry.WithMax(uint(option.RetryTimes())),                                // 最多三次
	}...))
	interceptors = append(interceptors, intercept.TimeoutUnaryClientInterceptor(option.Timeout(), option.Timeout()))
	interceptors = append(interceptors, intercept.ClientInterceptor(info))

	keepAlive := grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time: time.Duration(option.Keepalive()),
	})
	dialOptions := make([]grpc.DialOption, 0)

	dialOptions = append(dialOptions, grpc.WithBalancerName(option.BalanceName()))
	dialOptions = append(dialOptions, keepAlive)
	dialOptions = append(dialOptions, grpc.WithChainUnaryInterceptor(interceptors...))

	ctx := context.Background()
	cc, err := grpc.DialContext(ctx, "etcd:///"+option.ServiceName(), dialOptions...)

	if err != nil {
		log.Errorf(ctx, "dial grpc server %s", err)
	}
	log.Info(ctx, "start grpc client")
	return cc
}
