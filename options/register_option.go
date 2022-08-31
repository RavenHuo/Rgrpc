/**
 * @Author raven
 * @Description
 * @Date 2022/8/30
 **/
package options

const (
	defaultKeepAliveTtl = 10
)

type RegisterOptions struct {
	keepAliveTtl int
	endpoints    []string
}
type RegisterOption func(*RegisterOptions)

func WithRegisterKeepAliveTtl(ttl int) RegisterOption {
	return func(o *RegisterOptions) {
		o.keepAliveTtl = ttl
	}
}

func WithRegisterEndpoints(endpoints []string) RegisterOption {
	return func(o *RegisterOptions) {
		o.endpoints = endpoints
	}
}
func DefaultRegisterOption(ops ...RegisterOption) *RegisterOptions {
	defaultRegisterOptions := &RegisterOptions{
		keepAliveTtl: defaultKeepAliveTtl,
		endpoints:    []string{},
	}
	for _, opt := range ops {
		opt(defaultRegisterOptions)
	}
	return defaultRegisterOptions
}

func (r *RegisterOptions) Endpoints() []string {
	return r.endpoints
}
func (r *RegisterOptions) KeepAliveTtl() int {
	return r.keepAliveTtl
}
