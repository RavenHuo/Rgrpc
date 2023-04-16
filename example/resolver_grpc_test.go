/**
 * @Author raven
 * @Description
 * @Date 2022/9/7
 **/
package example

import (
	"context"
	"github.com/RavenHuo/Rgrpc/pb"
	"github.com/RavenHuo/Rgrpc/resolver"
	"github.com/RavenHuo/go-kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"testing"
	"time"
)

func TestResolver(t *testing.T) {
	// 注册服务发现组件
	grpcResolverBuilder := resolver.MustBuildSimpleBuilder("etcd", registerOption...)
	resolver.Register(grpcResolverBuilder)

	// etcd中注册5个服务
	go newServer(serverName, 1000)
	go newServer(serverName, 1001)
	go newServer(serverName, 1002)
	go newServer(serverName, 1003)
	go newServer(serverName, 1004)
	go newServer(serverName, 1005)
	time.Sleep(2 * time.Second)
	// 轮询的调用
	conn, err := grpc.Dial("etcd:///"+serverName, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		t.Fatalf("failed to dial %v", err)
	}
	defer conn.Close()

	// 进行十次数据请求
	for i := 0; i < 20; i++ {
		c := pb.NewGreeterClient(conn)
		resp, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: "raven"})
		if err != nil {
			t.Fatalf("say hello failed %v", err)
		}
		log.Infof(ctx, "grpc request success resp:%s", resp.Message)
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(30 * time.Second)
}
