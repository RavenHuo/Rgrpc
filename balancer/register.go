/**
 * @Author raven
 * @Description
 * @Date 2022/9/9
 **/
package balancer

import "google.golang.org/grpc/balancer"

func Register(balanceBuilder GrpcBalancerBuilder) {
	balancer.Register(balanceBuilder)
}
