/**
 * @Author raven
 * @Description
 * @Date 2022/9/6
 **/
package resolver

import (
	"context"
	"encoding/json"
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc/resolver"
	"sync"
)

type grpcResolverBuilder struct {
	// 服务名
	serverName string
	// 模式
	schema        string
	option        *options.RegisterOptions
	etcdClient    *etcd_client.Client
	closeCh       chan struct{}
	cc            resolver.ClientConn
	serverInfoSet map[string]*instance.ServerInfo
	// 更新的时候用锁来防止并发来更新
	mutex sync.Mutex
}

// impl Resolver resolver/resolver.go:248
func (s *grpcResolverBuilder) ResolveNow(nowOptions resolver.ResolveNowOptions) {
}

// impl Resolver resolver/resolver.go:248
func (s *grpcResolverBuilder) Close() {
	s.closeCh <- struct{}{}
}

func newSimpleBuilder(schema string, option ...options.RegisterOption) (*grpcResolverBuilder, error) {
	defaultOptions := options.DefaultRegisterOption(option...)
	return &grpcResolverBuilder{
		schema: schema,

		option:  defaultOptions,
		closeCh: make(chan struct{}),
		mutex:   sync.Mutex{},
	}, nil
}

// impl Builder resolver/resolver.go:227
// grpc服务发现的主方法
func (s *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// 这个EndPoint就是dial的时候etcd:///的(xxx)
	s.serverName = target.Endpoint
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: s.option.Endpoints()})
	if err != nil {
		log.Errorf(context.Background(), "build etcdClient failed err:%s", err)
		return nil, err
	}

	s.cc = cc
	s.etcdClient = etcdClient
	s.serverInfoSet = make(map[string]*instance.ServerInfo, 0)
	err = s.listen()
	if err != nil {
		log.Errorf(context.Background(), "ResolverBuilder listen failed err:%s", err)
		return nil, err
	}
	// keepAlive heartbeat and watch etcd key
	go s.keepAliveListen()
	return s, nil
}

// impl Builder resolver/resolver.go:227
func (s *grpcResolverBuilder) Scheme() string {
	return s.schema
}

// serverName:需要服务发现的服务名
func MustBuildSimpleBuilder(schema string, option ...options.RegisterOption) *grpcResolverBuilder {
	builder, err := newSimpleBuilder(schema, option...)
	if err != nil {
		log.Errorf(context.Background(), "new simpleBuilder err :%s", err)
		panic("new simpleBuilder failed")
	}
	return builder
}

func (s *grpcResolverBuilder) listen() error {
	serverInfoList, err := s.listenServerInfo()
	if err != nil {
		return err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, serverInfo := range serverInfoList {
		s.serverInfoSet[serverInfo.Key] = serverInfo
	}
	s.update()
	return nil
}

func (s *grpcResolverBuilder) listenServerInfo() ([]*instance.ServerInfo, error) {
	prefix := instance.GetServerPrefix(s.serverName)
	resp, err := s.etcdClient.GetDirectory(context.Background(), prefix)
	if err != nil {
		return nil, err
	}
	result := make([]*instance.ServerInfo, 0, len(resp))
	for k, v := range resp {
		var serverInfo instance.ServerInfo
		err := json.Unmarshal(v, &serverInfo)
		if err != nil {
			log.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, v, err)
		} else {
			serverInfo.Key = k
			result = append(result, &serverInfo)
		}
	}
	return result, nil
}

func (s *grpcResolverBuilder) keepAliveListen() {
	prefix := instance.GetServerPrefix(s.serverName)
	watchChan, err := s.etcdClient.WatchPrefix(context.Background(), prefix)
	if err != nil {
		log.Errorf(context.Background(), "watch %s err:%s", prefix, err)
	}

	for {
		select {
		// watch etcd prefix when update key
		case e := <-watchChan:
			func(e *etcd_client.Event) {
				s.mutex.Lock()
				defer s.mutex.Unlock()
				kv := e.Kv
				serverInfoList := s.serverInfoSet
				if e.Type == mvccpb.PUT {
					var serverInfo instance.ServerInfo
					err := json.Unmarshal(kv.Value, &serverInfo)
					if err != nil {
						log.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, string(kv.Key), err)
						return
					}
					serverInfoList[string(kv.Key)] = &serverInfo
					s.update()
				} else {
					// remove serverInfo in serverInfoSet
					delete(s.serverInfoSet, string(kv.Key))
					s.update()

				}
			}(e)
		// close
		case <-s.closeCh:
			log.Infof(context.Background(), "listen close channel stop keepAlive listen prefix:%s", prefix)
			return
		}
	}
}

func (s *grpcResolverBuilder) updateState() {
	state := resolver.State{}
	address := make([]resolver.Address, 0, len(s.serverInfoSet))
	serverInfoList := s.serverInfoSet
	for _, serverInfo := range serverInfoList {
		addr := instance.BuilderAddress(serverInfo)
		address = append(address, addr)
	}
	state.Addresses = address
	log.Infof(context.Background(), "update state :%+v", address)
	s.cc.UpdateState(state)
}

func (s *grpcResolverBuilder) update() {
	s.updateState()
}
