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

- Scheme() 方法返回的是使用的模式。需要跟grpc.Dial()方法的前缀匹配
- Buiild()方法是最关键的方法。
  - cc ClientConn（grpc链接情况），最终通过cc.UpdateState来修改服务调用地址
  - target Target（需要调用的远程服务名字），它是从用户传递给 Dial 或 DialContext 的目标字符串中解析出来的。 grpc 将其传递给解析器和平衡器。target.Endpoint 就是这个服务的名字

然后使用 resolver.Register() 方法将builder注册到grpc中

