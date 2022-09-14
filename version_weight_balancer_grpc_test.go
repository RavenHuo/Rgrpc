/**
 * @Author raven
 * @Description
 * @Date 2022/9/14
 **/
package grpc

import (
	"context"
	"encoding/json"
	"github.com/RavenHuo/grpc/balancer/version_weight"
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/pb"
	"github.com/RavenHuo/grpc/register"
	"github.com/RavenHuo/grpc/resolver"
	"github.com/RavenHuo/grpc/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVersionWeightBalancerGrpc(t *testing.T) {
	// 注册服务发现组件
	grpcResolverBuilder := resolver.MustBuildSimpleBuilder("etcd", defaultLogger, registerOption...)
	resolver.Register(grpcResolverBuilder)

	// etcd中注册5个服务
	go newServerWithVersion(serverName, 1000, "v0")
	go newServerWithVersion(serverName, 1001, "v1")
	go newServerWithVersion(serverName, 1002, "v2")
	go newServerWithVersion(serverName, 1003, "v3")
	go newServerWithVersion(serverName, 1004, "v4")
	go newServerWithVersion(serverName, 1005, "v5")
	time.Sleep(5 * time.Second)

	conn, err := grpc.Dial("etcd:///"+serverName, grpc.WithInsecure(), grpc.WithBalancerName(version_weight.Name))
	if err != nil {
		t.Fatalf("failed to dial %v", err)
	}
	defer conn.Close()

	versionMap := make(map[string]float64, 0)
	versionMap["v1"] = 10
	versionMap["v2"] = 50
	versionMap["v3"] = 60
	versionMap["v10"] = 60
	versionMapStr, _ := json.Marshal(versionMap)
	// 进行十次数据请求
	for i := 0; i < 20; i++ {
		c := pb.NewGreeterClient(conn)
		reqCtx := context.Background()
		md := metadata.MD{}
		md[instance.VersionWeightHeader] = []string{string(versionMapStr)}
		reqCtx = metadata.NewIncomingContext(reqCtx, md)

		resp, err := c.SayHello(reqCtx, &pb.HelloRequest{Name: "raven"})
		if err != nil {
			defaultLogger.Errorf(context.Background(), "say hello failed %v", err)
		}
		defaultLogger.Infof(ctx, "grpc request success resp:%s", resp.Message)
	}
}

func TestVersionWeightBalancerGrpcNotFound(t *testing.T) {
	// 注册服务发现组件
	grpcResolverBuilder := resolver.MustBuildSimpleBuilder("etcd", defaultLogger, registerOption...)
	resolver.Register(grpcResolverBuilder)

	// etcd中注册5个服务
	go newServerWithVersion(serverName, 1000, "v0")
	go newServerWithVersion(serverName, 1001, "v1")
	go newServerWithVersion(serverName, 1002, "v2")
	go newServerWithVersion(serverName, 1003, "v3")
	go newServerWithVersion(serverName, 1004, "v4")
	go newServerWithVersion(serverName, 1005, "v5")
	time.Sleep(5 * time.Second)

	conn, err := grpc.Dial("etcd:///"+serverName, grpc.WithInsecure(), grpc.WithBalancerName(version_weight.Name))
	if err != nil {
		t.Fatalf("failed to dial %v", err)
	}
	defer conn.Close()

	versionMap := make(map[string]float64, 0)
	versionMap["v10"] = 60
	versionMapStr, _ := json.Marshal(versionMap)
	// 进行十次数据请求
	for i := 0; i < 20; i++ {
		c := pb.NewGreeterClient(conn)
		reqCtx := context.Background()
		md := metadata.MD{}
		md[instance.VersionWeightHeader] = []string{string(versionMapStr)}
		reqCtx = metadata.NewIncomingContext(reqCtx, md)

		resp, err := c.SayHello(reqCtx, &pb.HelloRequest{Name: "raven"})
		if err != nil {
			defaultLogger.Errorf(context.Background(), "say hello failed %v", err)
			if !strings.Contains(err.Error(), version_weight.ErrNoMatchVersionConn.Error()) {
				t.Fatalf("not found test failed %v", err)
			}
			continue
		}
		defaultLogger.Infof(ctx, "grpc request success resp:%s", resp.Message)
	}
}

func newServerWithVersion(serverName string, port int, version string) {
	r, err := register.NewRegister(defaultLogger, registerOption...)
	if err != nil {
		defaultLogger.Errorf(ctx, "register failed")
		return
	}
	defer r.Unregister()

	listen, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		defaultLogger.Errorf(ctx, "failed to listen %v", err)
		return
	}

	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &VersionServer{Port: port, Version: version})
	mateData := make(map[string]interface{}, 0)
	mateData[instance.Version] = version
	info := &instance.ServerInfo{
		Name:     serverName,
		Port:     port,
		MateData: mateData,
	}

	if err := r.Register(info); err != nil {
		defaultLogger.Errorf(ctx, "register failed")
		return
	}

	if err := s.Serve(listen); err != nil {
		defaultLogger.Errorf(ctx, "failed to server %v", err)
	}
}

type VersionServer struct {
	Port    int
	Version string
} //服务对象

// SayHello 实现服务的接口 在proto中定义的所有服务都是接口
func (s *VersionServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	defaultLogger.Infof(ctx, "say hello port:%d ,version:%s, reqVersion:%s", s.Port, s.Version, utils.GetGrpcHeader(ctx, instance.VersionWeightHeader))
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}