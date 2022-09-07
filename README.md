# grpc

include grpc register and discovery



## grpc的服务注册与发现

大概流程如下

![image](https://s2.loli.net/2022/09/05/Eb2edCGcDNhVXAa.png)

grpc框架在源码中定义了两个重要的接口，Builder及Resolver接口

### Resolver接口

resolver/resolver.go:248

```go
// Resolver watches for the updates on the specified target.
// Updates include address updates and service config updates.
//解析器监视指定目标的更新。
//更新包括地址更新和服务配置更新。
type Resolver interface {
   // ResolveNow will be called by gRPC to try to resolve the target name
   // again. It's just a hint, resolver can ignore this if it's not necessary.
   // gRPC将调用ResolveNow来解析目标名称
   //再次。这只是一个提示，如果不需要，解析器可以忽略它。
     
   // It could be called multiple times concurrently.
   //它可以并发调用多次。
   ResolveNow(ResolveNowOptions)
   // Close closes the resolver.
   Close()
}
```

Resolver接口是一个解析器，是服务发现返回的obj,因为接口需要返回这个通用对象。所以这个Resolver的Close()方法才是最重要的。

- ResolveNow，需要发现的服务选项
- Close() 关闭服务发现



### Builder接口

resolver/resolver.go:227

```go
// Builder creates a resolver that will be used to watch name resolution updates.
//生成器创建一个用于监视名称解析更新的解析程序。
type Builder interface {
   // Build creates a new resolver for the given target.
   // gRPC dial calls Build synchronously, and fails if the returned error is not nil
    
   //生成为给定目标创建一个新的解析器。
   //gRPC拨号呼叫同步生成，如果返回的错误不为空
   Build(target Target, cc ClientConn, opts BuildOptions) (Resolver, error)
   // Scheme returns the scheme supported by this resolver.
   //Scheme返回此解析器支持的方案。
   // Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
   Scheme() string
}
```

builder是一个grpc框架暴露出来的服务发现接口

- Scheme() 方法返回的是服务的名字。需要跟grpc.Dial()方法的前缀匹配
- Buiild()方法，最终通过修改cc resolver.ClientConn 来修改服务调用地址。

然后使用 resolver.Register() 方法将builder注册到grpc中

### Builder实现

我们来看看 Builder 接口的具体实现

```go
type baseBuilder struct {
    name string
    reg  registry.Registry
}

// Build ...
func (b *baseBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
    endpoints, err := b.reg.WatchServices(context.Background(), target.Endpoint, "grpc")
    if err != nil {
        return nil, err
    }

    var stop = make(chan struct{})
    xgo.Go(func() {
        for {
            select {
            case endpoint := <-endpoints:
                var state = resolver.State{
                    Addresses: make([]resolver.Address, 0),
                      ...
                }
                for _, node := range endpoint.Nodes {
                    ...
                    state.Addresses = append(state.Addresses, address)
                }
                cc.UpdateState(state)
            case <-stop:
                return
            }
        }
    })

    return &baseResolver{
        stop: stop,
    }, nil
}
```

这里Build 方法主要是通过 Registry 模块获得监控服务通道，然后将更新的服务信息再更新到 grpcClient 中去，保证 grpcClient 的负载均衡器的服务地址永远都是最新的。

如何将Builder的具体实现注册到 grpc 中。

```go
import "google.golang.org/grpc/resolver"

// Register ...
func Register(name string, reg registry.Registry) {
    resolver.Register(&baseBuilder{
        name: name,
        reg:  reg,
    })
}
```

将 Registry模块注入到 Builder 对象中，然后注入到 grpc 的 resolver 模块中去。这样 grpcClient 在实际运行中就会调用 etcd 的服务发现功能了。