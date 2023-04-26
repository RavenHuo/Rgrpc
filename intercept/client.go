/**
 * @Author raven
 * @Description
 * @Date 2023/4/17
 **/
package intercept

import (
	"context"
	"fmt"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/go-kit/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"time"
)

const ClientField = "client"

// ClientInterceptor 客户端请求拦截器 用于透传TractId
func ClientInterceptor(info *instance.ServerInfo) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		client := fmt.Sprintf("trace:%s:%d", info.Name, info.Port)

		ctx = metadata.AppendToOutgoingContext(ctx, ClientField, client, trace.TraceIdField, trace.GetTraceId(ctx))

		// Logic before invoking the invoker
		log.Infof(ctx, "#grpc# client start request %s Req:[%+v]", method, req)

		// Calls the invoker to execute RPC
		err := invoker(ctx, method, req, reply, cc, opts...)
		// Logic after invoking the invoker

		log.Infof(ctx, "#grpc# client finish request %s Req:[%+v] Resp:[:%+v] in %.3fms: error=%s",
			method, req, reply, float32(time.Since(start))/float32(time.Millisecond), errStr(err))

		return err
	}
}

// TimeoutUnaryClientInterceptor gRPC客户端超时拦截器
func TimeoutUnaryClientInterceptor(timeout time.Duration, slowThreshold time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		now := time.Now()
		// 若无自定义超时设置，默认设置超时
		_, ok := ctx.Deadline()
		if !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		du := time.Since(now)

		if slowThreshold > time.Duration(0) && du > slowThreshold {
			log.Errorf(ctx, "#grpc# client finish request %s Req:[%+v] Resp:[:%+v] in %.3fms: error=%s",
				method, req, reply, float32(time.Since(now))/float32(time.Millisecond), errStr(err))
		}
		return err
	}
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
