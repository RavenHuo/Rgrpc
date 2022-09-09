/**
 * @Author raven
 * @Description
 * @Date 2022/9/9
 **/
package balancer

import "google.golang.org/grpc/balancer"

type GrpcBalancerBuilder struct {
}

func (g GrpcBalancerBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	panic("implement me")
}

func (g GrpcBalancerBuilder) Name() string {
	panic("implement me")
}
