[TOC]



## grpc

include grpc register and discovery



## 1、grpc的服务注册与发现

大概流程如下

![image](https://s2.loli.net/2022/09/05/Eb2edCGcDNhVXAa.png)

grpc框架在源码中定义了两个重要的接口，Builder及Resolver接口

### 1.1、Resolver接口

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



### 1.2、Builder接口

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

然后使用 resolver.Register() 方法将builder注册到grpc中。



### 1.3、服务发现的编码流程

- 定义builder（实现 resolver.Builder，实现 resolver.Resolver）
- 调用 resolver.Register 注册自定义的 Resolver，其中 name 为 target 中的 scheme
- 在Builder.Builder() 方法中实现服务发现逻辑 (etcd、consul、zookeeper)
- 通过 resolver.ClientConn 实现服务地址的更新



### 1.4、grpc调用流程

那么grpc框架调用的流程是怎样的呢？

resolver/resolver.go:47

```
var (
   m = make(map[string]Builder)
   defaultScheme = "passthrough"
)
func Register(b Builder) {
   m[b.Scheme()] = b
}
func Get(scheme string) Builder {
	if b, ok := m[scheme]; ok {
		return b
	}
	return nil
}
```

resolver.Register()方法将builder注入到grpc的map中，key是scheme（模式）



### 1.5、grpc.DialContext()

创建grpc链接,调用DialContext()方法来处理		

```go
func Dial(target string, opts ...DialOption) (*ClientConn, error) {
   return DialContext(context.Background(), target, opts...)
}
```

![img](https://s2.loli.net/2022/09/08/nAy8LmgGe3FWsdT.jpg)

```go
func DialContext(ctx context.Context, target string, opts ...DialOption) (conn *ClientConn, err error) {
   cc := &ClientConn{
      target:            target,
      csMgr:             &connectivityStateManager{},
      conns:             make(map[*addrConn]struct{}),
      dopts:             defaultDialOptions(),
      blockingpicker:    newPickerWrapper(),
      czData:            new(channelzData),
      firstResolveEvent: grpcsync.NewEvent(),
   }
   cc.retryThrottler.Store((*retryThrottler)(nil))
   cc.ctx, cc.cancel = context.WithCancel(context.Background())

   for _, opt := range opts {
      opt.apply(&cc.dopts)
   }
   // 拦截器	
   chainUnaryClientInterceptors(cc)
   chainStreamClientInterceptors(cc)
   ...
   if cc.dopts.resolverBuilder == nil {
      // 解析target 
      cc.parsedTarget = parseTarget(cc.target)
      grpclog.Infof("parsed scheme: %q", cc.parsedTarget.Scheme)
      // getResolver，获取可用的服务发现组件 
      cc.dopts.resolverBuilder = resolver.Get(cc.parsedTarget.Scheme)
      if cc.dopts.resolverBuilder == nil {
         grpclog.Infof("scheme %q not registered, fallback to default scheme", cc.parsedTarget.Scheme)
         cc.parsedTarget = resolver.Target{
            Scheme:   resolver.GetDefaultScheme(),
            Endpoint: target,
         }
         // getResolver，获取可用的服务发现组件 
         cc.dopts.resolverBuilder = resolver.Get(cc.parsedTarget.Scheme)
      }
   } else {
      cc.parsedTarget = resolver.Target{Endpoint: target}
   }
   creds := cc.dopts.copts.TransportCredentials
   if creds != nil && creds.Info().ServerName != "" {
      cc.authority = creds.Info().ServerName
   } else if cc.dopts.insecure && cc.dopts.authority != "" {
      cc.authority = cc.dopts.authority
   } else {
      // Use endpoint from "scheme://authority/endpoint" as the default
      // authority for ClientConn.
      cc.authority = cc.parsedTarget.Endpoint
   }

   var credsClone credentials.TransportCredentials
   if creds := cc.dopts.copts.TransportCredentials; creds != nil {
      credsClone = creds.Clone()
   }
   cc.balancerBuildOpts = balancer.BuildOptions{
      DialCreds:        credsClone,
      CredsBundle:      cc.dopts.copts.CredsBundle,
      Dialer:           cc.dopts.copts.Dialer,
      ChannelzParentID: cc.channelzID,
      Target:           cc.parsedTarget,
   }

   // 包装Resolver
   rWrapper, err := newCCResolverWrapper(cc)
   if err != nil {
      return nil, fmt.Errorf("failed to build resolver: %v", err)
   }

   cc.mu.Lock()
   cc.resolverWrapper = rWrapper
   cc.mu.Unlock()
   // A blocking dial blocks until the clientConn is ready.
   if cc.dopts.block {
      for {
         s := cc.GetState()
         if s == connectivity.Ready {
            break
         } else if cc.dopts.copts.FailOnNonTempDialError && s == connectivity.TransientFailure {
            if err = cc.blockingpicker.connectionError(); err != nil {
               terr, ok := err.(interface {
                  Temporary() bool
               })
               if ok && !terr.Temporary() {
                  return nil, err
               }
            }
         }
         if !cc.WaitForStateChange(ctx, s) {
            // ctx got timeout or canceled.
            return nil, ctx.Err()
         }
      }
   }

   return cc, nil
}
```

步骤如下：

- 1、多拦截器处理

  ```go
  chainUnaryClientInterceptors(cc)
  chainStreamClientInterceptors(cc)
  ```

  分别有两个拦截器，一个是短连接的拦截器，一个是stream流的拦截器。

- 2、cc.parsedTarget = parseTarget(cc.target) 解析target。

  ```go
  func parseTarget(target string) (ret resolver.Target) {
     var ok bool
     ret.Scheme, ret.Endpoint, ok = split2(target, "://")
     if !ok {
        return resolver.Target{Endpoint: target}
     }
     ret.Authority, ret.Endpoint, ok = split2(ret.Endpoint, "/")
     if !ok {
        return resolver.Target{Endpoint: target}
     }
     return ret
  }
  ```

  在这里会将dial中传递的target转化成resolver.Target对象（也就是builder中的Target对象），并且在这里会将target拆解成Target对象的scheme和endpoint，**假如我们target传的是etcd:///hello-service，那么这里就会解析成scheme为etcd，endpoint为hello-service**

- 3、cc.dopts.resolverBuilder = resolver.Get(cc.parsedTarget.Scheme) 获取服务发现注册组件。

  调用时就会调用resolver.Get()方法，从第一步注册服务发现组件的map中获取	

- 4、 rWrapper, err := newCCResolverWrapper(cc)

```go
func newCCResolverWrapper(cc *ClientConn) (*ccResolverWrapper, error) {
   rb := cc.dopts.resolverBuilder
   if rb == nil {
      return nil, fmt.Errorf("could not get resolver for scheme: %q", cc.parsedTarget.Scheme)
   }

   ccr := &ccResolverWrapper{
      cc:   cc,
      done: grpcsync.NewEvent(),
   }
    rbo := resolver.BuildOptions{
        DisableServiceConfig: cc.dopts.disableServiceConfig,
        DialCreds:            credsClone,
        CredsBundle:          cc.dopts.copts.CredsBundle,
        Dialer:               cc.dopts.copts.Dialer,
    }
   // ...
   // 在这里执行Build方法
   ccr.resolver, err = rb.Build(cc.parsedTarget, ccr, rbo)
   if err != nil {
      return nil, err
   }
   return ccr, nil
}
```

包装注册进来的Builder类，然后将解析好的Target对象，及包装过的grpc.ClientConn,负载均衡选项也一起作为入参，来执行用户自定义的Builder.Build()方法。

### 1.6、注意

值得注意的是，每调用一次DialContext方法，就会调用一次newCCResolverWrapper方法去包装grpcClient，所以我们调用grpc的时候实际上只需要调用一次grpc.Dial()方法就好了，grpc框架内部是自带连接池的。



### 1.7、UpdateState

(ccr *ccResolverWrapper) UpdateState(s resolver.State) 更新grpcClient的地址

/resolver_conn_wrapper.go:174

在Builder.Build()方法里面，当链接地址发生变更的时候，我们通常使用UpdateState(state)来更新地址。



```go
func (ccr *ccResolverWrapper) UpdateState(s resolver.State) {
   ...
   ccr.curState = s
   ccr.poll(ccr.cc.updateResolverState(ccr.curState, nil))
}

func (cc *ClientConn) updateResolverState(s resolver.State, err error) error {
	var ret error
     // 判断是否使用默认的配置
	if cc.dopts.disableServiceConfig || s.ServiceConfig == nil {
		cc.maybeApplyDefaultServiceConfig(s.Addresses)
	} else {// 使用用户自定义的配置
		if sc, ok := s.ServiceConfig.Config.(*ServiceConfig); s.ServiceConfig.Err == nil && ok {
            // 加载负载均衡器
			cc.applyServiceConfigAndBalancer(sc, s.Addresses)
		} else {
            // 加载失败
			ret = balancer.ErrBadResolverState
			if cc.balancerWrapper == nil {
				cc.blockingpicker.updatePicker(base.NewErrPicker(err))
				cc.csMgr.updateState(connectivity.TransientFailure)
                // 注意：这里return了，意味着自定义serviceConfig的话，就不能 updateClientConnState了
				return ret
			}
		}
	}

	var balCfg serviceconfig.LoadBalancingConfig
	if cc.dopts.balancerBuilder == nil && cc.sc != nil && cc.sc.lbConfig != nil {
		balCfg = cc.sc.lbConfig.cfg
	}

    // 以下是负载均衡相关的
	cbn := cc.curBalancerName
	bw := cc.balancerWrapper
	cc.mu.Unlock()
	if cbn != grpclbName {
		// Filter any grpclb addresses since we don't have the grpclb balancer.
		for i := 0; i < len(s.Addresses); {
			if s.Addresses[i].Type == resolver.GRPCLB {
				copy(s.Addresses[i:], s.Addresses[i+1:])
				s.Addresses = s.Addresses[:len(s.Addresses)-1]
				continue
			}
			i++
		}
	}
    // 真正的更新update
	uccsErr := bw.updateClientConnState(&balancer.ClientConnState{ResolverState: s, BalancerConfig: balCfg})
	if ret == nil {
		ret = uccsErr // prefer ErrBadResolver state since any other error is
		// currently meaningless to the caller.
	}
	return ret
}
```

更新的时候首先判断grpc服务配置是否为空，为空的时候就会调用maybeApplyDefaultServiceConfig()方法，执行默认配置。有配置的情况下会执行applyServiceConfigAndBalancer()方法来执行配置及负载均衡器。

然后配置负载均衡策略，最后才是真正执行**updateClientConnState，用于更新链接状态及配置负载均衡策略**



### 1.8、applyServiceConfigAndBalancer

```go
func (cc *ClientConn) applyServiceConfigAndBalancer(sc *ServiceConfig, addrs []resolver.Address) {
	if cc.dopts.balancerBuilder == nil {
		var newBalancerName string
		if cc.sc != nil && cc.sc.lbConfig != nil {
			newBalancerName = cc.sc.lbConfig.name
		} else {
			var isGRPCLB bool
			for _, a := range addrs {
				if a.Type == resolver.GRPCLB {
					isGRPCLB = true
					break
				}
			}
			if isGRPCLB {
				newBalancerName = grpclbName
			} else if cc.sc != nil && cc.sc.LB != nil {
				newBalancerName = *cc.sc.LB
			} else {
                // 默认的负载均衡策略是pick_fist
				newBalancerName = PickFirstBalancerName
			}
		}
		cc.switchBalancer(newBalancerName)
	} else if cc.balancerWrapper == nil {
		cc.curBalancerName = cc.dopts.balancerBuilder.Name()
		cc.balancerWrapper = newCCBalancerWrapper(cc, cc.dopts.balancerBuilder, cc.balancerBuildOpts)
	}
}

```

就在这个方法里面去加载负载均衡策略，假如balancerBuilder没有配置的话，默认的负载均衡策略是pick_fist。balancerBuilder假如配置了的话就会将自定义的balance用newCCBalancerWrapper()包装一下。这个balancerBuilder是一个负载均衡接口，。



## 2、负载均衡



![img](https://s2.loli.net/2022/09/09/wTm9dFaubMjtpZo.jpg)

到此 ClientConn 创建过程基本结束，我们再一起梳理一下整个过程，首先获取 resolver，其中 ccResolverWrapper 实现了 resovler.ClientConn 接口，通过 Resolver 的 UpdateState 方法触发获取 Balancer，获取 Balancer，其中 ccBalancerWrapper 实现了 balancer.ClientConn 接口，通过 Balnacer 的 UpdateClientConnState 方法触发创建连接 (SubConn)，最后创建 HTTP2 Client