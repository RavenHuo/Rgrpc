/**
 * @Author raven
 * @Description
 * @Date 2022/9/13
 **/
package balancer

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

type GrpcPicker struct {
}

func (g GrpcPicker) Build(readySCs map[resolver.Address]balancer.SubConn) balancer.Picker {
	panic("implement me")
}

