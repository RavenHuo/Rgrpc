/**
 * @Author raven
 * @Description
 * @Date 2022/9/2
 **/
package grpc

import (
	"context"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/grpc/discovery"
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/options"
	"github.com/RavenHuo/grpc/pb"
	"github.com/RavenHuo/grpc/register"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"net"
	"strconv"
	"testing"
	"time"
)

var etcdAddrs = []string{"localhost:2379"}
var registerOption = []options.GrpcOption{
	options.WithEndpoints(etcdAddrs),
	options.WithKeepAliveTtl(10),
}
var serverName = "hello-service"

var ctx = context.Background()

func TestDiscovery(t *testing.T) {
	log.UseLogger(log.BuildTraceLogger())
	// etcd中注册5个服务
	go newServer(serverName, 1000)
	go newServer(serverName, 1001)
	go newServer(serverName, 1002)
	go newServer(serverName, 1003)
	go newServer(serverName, 1004)
	go newServer(serverName, 1005)
	time.Sleep(2 * time.Second)
	disc, err := discovery.NewDiscovery(registerOption...)
	if err != nil {
		log.Errorf(ctx, "disc err:%s", err)
	}
	err = disc.Listen(serverName)
	if err != nil {
		return
	}
	serverList := disc.ListServerInfo(serverName)

	log.Infof(ctx, "disc serverInfo List :%v", serverList)
	// 进行十次数据请求
	for i := 0; i < 20; i++ {
		serverInfo := serverList[i%6]
		conn, err := grpc.Dial(serverInfo.FullAddress(), grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
		c := pb.NewGreeterClient(conn)
		if err != nil {
			t.Fatalf("failed to dial %v", err)
		}
		defer conn.Close()

		resp, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: "raven"})
		if err != nil {
			t.Fatalf("say hello failed %v", err)
		}
		log.Infof(ctx, "grpc request success resp:%s", resp.Message)
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)
}

func newServer(serverName string, port int) {
	r, err := register.NewRegister(registerOption...)
	if err != nil {
		log.Errorf(ctx, "register failed")
		return
	}
	defer r.Unregister()

	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Errorf(ctx, "failed to listen %v", err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{Port: port})

	info := &instance.ServerInfo{
		Name: serverName,
		Port: port,
	}

	if err := r.Register(info); err != nil {
		log.Errorf(ctx, "register failed")
		return
	}

	if err := s.Serve(listen); err != nil {
		log.Errorf(ctx, "failed to server %v", err)
	}
}

type server struct {
	Port int
} //服务对象

// SayHello 实现服务的接口 在proto中定义的所有服务都是接口
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Infof(ctx, "say hello port:%d", s.Port)
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}
