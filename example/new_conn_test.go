package example

import (
	"context"
	"github.com/RavenHuo/Rgrpc"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/Rgrpc/pb"
	"github.com/RavenHuo/Rgrpc/resolver"
	"github.com/RavenHuo/go-kit/log"
	"testing"
	"time"
)

func TestNewConn(t *testing.T) {
	// etcd中注册5个服务
	go newServer(serverName, 1000)
	go newServer(serverName, 1001)
	go newServer(serverName, 1002)
	go newServer(serverName, 1003)
	go newServer(serverName, 1004)
	go newServer(serverName, 1005)
	time.Sleep(2 * time.Second)
	serveInfo := &instance.ServerInfo{
		Name:     "hello-server",
		Port:     10086,
		MateData: nil,
	}
	// 服务发现
	grpcResolverBuilder := resolver.MustBuildSimpleBuilder("etcd", registerOption...)
	resolver.Register(grpcResolverBuilder)
	callOption := make([]options.CallOption, 0)
	callOption = append(callOption, options.WithServiceName(serverName))
	grcConn := Rgrpc.NewResolveConn(serveInfo, callOption)
	defer grcConn.Close()

	// 进行十次数据请求
	for i := 0; i < 20; i++ {
		resp, err := pb.NewGreeterClient(grcConn).SayHello(context.Background(), &pb.HelloRequest{Name: "raven"})
		if err != nil {
			t.Fatalf("say hello failed %v", err)
		}
		log.Infof(ctx, "grpc request success resp:%s", resp.Message)
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(30 * time.Second)
}
