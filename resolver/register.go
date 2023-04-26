/**
 * @Author raven
 * @Description
 * @Date 2022/9/7
 **/
package resolver

import "google.golang.org/grpc/resolver"

// 注册服务发现组件
func Register(s *GrpcResolverBuilder) {
	resolver.Register(s)
}
