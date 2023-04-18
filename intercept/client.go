/**
 * @Author raven
 * @Description
 * @Date 2023/4/17
 **/
package intercept

import (
	"context"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/go-kit/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
	"strconv"
	"time"
)

const ClientField = "client"

// ClientInterceptor 客户端请求拦截器 用于透传TractId
func ClientInterceptor(ctx context.Context, method string, req interface{}, reply interface{}, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {
	start := time.Now()
	hostName, _ := os.Hostname()
	pid := os.Getpid()
	client := hostName + ":" + strconv.Itoa(pid)

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

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
