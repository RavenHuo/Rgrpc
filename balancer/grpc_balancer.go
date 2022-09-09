/**
 * @Author raven
 * @Description
 * @Date 2022/9/9
 **/
package balancer

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/resolver"
)

type GrpcBalancer struct {

}

func (g GrpcBalancer) HandleSubConnStateChange(sc balancer.SubConn, state connectivity.State) {
	panic("implement me")
}

func (g GrpcBalancer) HandleResolvedAddrs(addresses []resolver.Address, err error) {
	panic("implement me")
}

func (g GrpcBalancer) Close() {
	panic("implement me")
}

