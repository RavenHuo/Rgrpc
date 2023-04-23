package example

import (
	"github.com/RavenHuo/Rgrpc"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/Rgrpc/pb"
	"google.golang.org/grpc"
	"testing"
)

func TestNewServer(t *testing.T) {

	registerOption := []options.RegisterOption{
		options.WithEndpoints(etcdAddrs),
		options.WithKeepAliveTtl(10),
	}
	// 服务信息
	serveInfo := &instance.ServerInfo{
		Name:     "hello-service",
		Port:     8080,
		MateData: nil,
	}

	grpcFunc := func(s *grpc.Server) {
		pb.RegisterGreeterServer(s, &VersionServer{Port: 8080, Version: "1"})
	}
	// 创建grpc服务
	grpcServer := Rgrpc.NewGrpcServer(serveInfo, nil, grpcFunc)
	// 注册
	err := grpcServer.Register(registerOption...)
	if err != nil {
		panic(err)
	}
	grpcServer.Serve()
}
