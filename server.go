/**
 * @Author raven
 * @Description
 * @Date 2023/4/18
 **/
package Rgrpc

import (
	"context"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/intercept"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/Rgrpc/register"
	"github.com/RavenHuo/go-kit/log"
	"google.golang.org/grpc"
	"net"
	"strconv"
)

type Server struct {
	*grpc.Server
	listener net.Listener
	info     *instance.ServerInfo
	register *register.Register
}

func BuildGrpcServer(info *instance.ServerInfo, opts []options.GrpcOption, regFunc func(*grpc.Server)) *Server {

	ctx := context.Background()
	log.Infof(ctx, "listen at port: %d", info.Port)

	opt := &options.GrpcOptions{}
	for _, o := range opts {
		o(opt)
	}
	serverOptions := opt.ServerOptions()
	unaryInterceptor := make([]grpc.UnaryServerInterceptor, 0)
	unaryInterceptor = append(unaryInterceptor, intercept.TraceInterceptor)
	unaryInterceptor = append(unaryInterceptor, intercept.UnaryInterceptor)
	serverOptions = append(serverOptions,
		grpc.StreamInterceptor(intercept.StreamInterceptorChain(opt.StreamInterceptors()...)),
		grpc.UnaryInterceptor(intercept.UnaryInterceptorChain(unaryInterceptor...)),
	)
	grpcServer := grpc.NewServer(serverOptions...)
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(info.Port))
	if err != nil {
		panic("new grpc server err " + err.Error())
	}

	server := &Server{
		Server:   grpcServer,
		listener: listener,
		info:     info,
	}
	regFunc(server.Server)
	return server
}

func (s *Server) Register(opts ...options.RegisterOption) error {
	r, err := register.NewRegister(opts...)
	if err != nil {
		return err
	}
	s.register = r
	return s.register.Register(s.info)
}

// Server implements server.Server interface.
func (s *Server) Serve() error {
	err := s.Server.Serve(s.listener)
	return err
}

// Stop implements server.Server interface
// it will terminate echo server immediately
func (s *Server) Stop() error {
	s.Server.Stop()
	return nil
}

// GracefulStop implements server.Server interface
// it will stop echo server gracefully
func (s *Server) GracefulStop(ctx context.Context) error {
	if s.register != nil {
		if err := s.register.Unregister(); err != nil {
			return err
		}
	}
	s.Server.GracefulStop()
	return nil
}
