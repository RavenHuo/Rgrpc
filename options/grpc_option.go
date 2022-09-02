/**
 * @Author raven
 * @Description
 * @Date 2022/8/30
 **/
package options

const (
	defaultKeepAliveTtl = 10
)

type GrpcOptions struct {
	keepAliveTtl int
	endpoints    []string
}
type GrpcOption func(*GrpcOptions)

func WithKeepAliveTtl(ttl int) GrpcOption {
	return func(o *GrpcOptions) {
		o.keepAliveTtl = ttl
	}
}

func WithEndpoints(endpoints []string) GrpcOption {
	return func(o *GrpcOptions) {
		o.endpoints = endpoints
	}
}
func DefaultRegisterOption(ops ...GrpcOption) *GrpcOptions {
	defaultRegisterOptions := &GrpcOptions{
		keepAliveTtl: defaultKeepAliveTtl,
		endpoints:    []string{},
	}
	for _, opt := range ops {
		opt(defaultRegisterOptions)
	}
	return defaultRegisterOptions
}

func (r *GrpcOptions) Endpoints() []string {
	return r.endpoints
}
func (r *GrpcOptions) KeepAliveTtl() int {
	return r.keepAliveTtl
}
