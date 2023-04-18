/**
 * @Author raven
 * @Description
 * @Date 2023/4/17
 **/
package intercept

import (
	"context"
	"github.com/RavenHuo/Rgrpc/utils"
	"github.com/RavenHuo/go-kit/trace"
	"google.golang.org/grpc"
)

// TraceInterceptor 服务端的拦截器，用于解析tractId
func TraceInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	traceId := utils.GetGrpcHeader(ctx, trace.TraceIdField)
	client := utils.GetGrpcHeader(ctx, ClientField)

	grpcCtx := context.Background()
	grpcCtx = trace.SetTraceId(grpcCtx, traceId)
	grpcCtx = context.WithValue(grpcCtx, ClientField, client)
	resp, err = handler(grpcCtx, req)
	return resp, err
}
