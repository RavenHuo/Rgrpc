/**
 * @Author raven
 * @Description
 * @Date 2022/9/6
 **/
package resolver

import (
	"context"
	"encoding/json"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/options"
	"go.etcd.io/etcd/api/v3/mvccpb"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

type grpcResolverBuilder struct {
	// 服务名
	serverName string
	// 模式
	schema         string
	option         *options.GrpcOptions
	etcdClient     *etcd_client.Client
	logger         log.ILogger
	closeCh        chan struct{}
	cc             resolver.ClientConn
	serverInfoList []*instance.ServerInfo
	rwMutex        sync.RWMutex
}

// impl Resolver resolver/resolver.go:248
func (s *grpcResolverBuilder) ResolveNow(nowOptions resolver.ResolveNowOptions) {
}

// impl Resolver resolver/resolver.go:248
func (s *grpcResolverBuilder) Close() {
	s.closeCh <- struct{}{}
}

func newSimpleBuilder(schema string, logger log.ILogger, option ...options.GrpcOption) (*grpcResolverBuilder, error) {
	defaultOptions := options.DefaultRegisterOption(option...)
	return &grpcResolverBuilder{
		schema:  schema,
		option:  defaultOptions,
		logger:  logger,
		rwMutex: sync.RWMutex{},
	}, nil
}

// impl Builder resolver/resolver.go:227
// grpc服务发现的主方法
func (s *grpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	// 这个EndPoint就是dial的时候etcd:///的(xxx)
	s.serverName = target.Endpoint
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: s.option.Endpoints()}, s.logger)
	if err != nil {
		s.logger.Errorf(context.Background(), "build etcdClient failed err:%s", err)
		return nil, err
	}

	s.cc = cc
	s.etcdClient = etcdClient
	err = s.listen()
	if err != nil {
		s.logger.Errorf(context.Background(), "ResolverBuilder listen failed err:%s", err)
		return nil, err
	}
	return s, nil
}

// impl Builder resolver/resolver.go:227
func (s *grpcResolverBuilder) Scheme() string {
	return s.schema
}

// serverName:需要服务发现的服务名
func MustBuildSimpleBuilder(schema string, logger log.ILogger, option ...options.GrpcOption) *grpcResolverBuilder {
	builder, err := newSimpleBuilder(schema, logger, option...)
	if err != nil {
		logger.Errorf(context.Background(), "new simpleBuilder err :%s", err)
		panic("new simpleBuilder failed")
	}
	return builder
}

func (s *grpcResolverBuilder) listen() error {
	serverIndoList, err := s.listenServerInfo()
	if err != nil {
		return err
	}
	s.update(serverIndoList)
	// keepAlive heartbeat and watch etcd key
	go s.keepAliveListen()
	return nil
}

func (s *grpcResolverBuilder) listenServerInfo() ([]*instance.ServerInfo, error) {
	prefix := instance.BuildServerPrefix(s.serverName)
	resp, err := s.etcdClient.GetDirectory(context.Background(), prefix)
	if err != nil {
		return nil, err
	}
	result := make([]*instance.ServerInfo, 0, len(resp))
	for _, v := range resp {
		var serverInfo instance.ServerInfo
		err := json.Unmarshal(v, &serverInfo)
		if err != nil {
			s.logger.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, v, err)
		} else {
			result = append(result, &serverInfo)
		}
	}
	return result, nil
}

func (s *grpcResolverBuilder) keepAliveListen() {
	timer := time.NewTimer(time.Duration(s.option.KeepAliveTtl()) * time.Second)
	prefix := instance.BuildServerPrefix(s.serverName)
	watchChan, err := s.etcdClient.WatchPrefix(context.Background(), prefix)
	if err != nil {
		s.logger.Errorf(context.Background(), "watch %s err:%s", prefix, err)
	}

	for {
		select {
		// timer update serverMap
		case <-timer.C:
			serverInfoList, err := s.listenServerInfo()
			s.logger.Infof(context.Background(), "heartbeat to listen etcd serverName:%s", s.serverName)
			if err == nil {
				s.update(serverInfoList)
			}
		// watch etcd prefix when update key
		case e := <-watchChan:
			kv := e.Kv
			serverInfoList := s.listServerInfo()
			if e.Type == mvccpb.PUT {
				var serverInfo instance.ServerInfo
				err := json.Unmarshal(kv.Value, &serverInfo)
				if err != nil {
					s.logger.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, string(kv.Key), err)
					continue
				}
				serverInfoList = append(serverInfoList, &serverInfo)
				s.update(serverInfoList)
			} else {
				index := -1
				for i, s := range serverInfoList {
					if s.Key == string(kv.Key) {
						index = i
					}
				}
				if index != -1 {
					// remove serverInfo in serverInfoList
					serverInfoList = append(serverInfoList[:index], serverInfoList[index+1:]...)
					s.update(serverInfoList)
				}
			}

		// close
		case <-s.closeCh:
			return
		}
	}
}

func (s *grpcResolverBuilder) updateServerInfoList(serverInfoList []*instance.ServerInfo) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	s.serverInfoList = serverInfoList
}

func (s *grpcResolverBuilder) listServerInfo() []*instance.ServerInfo {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.serverInfoList
}

func (s *grpcResolverBuilder) updateState() {
	state := resolver.State{}
	address := make([]resolver.Address, 0, len(s.serverInfoList))
	serverInfoList := s.listServerInfo()
	for _, serverInfo := range serverInfoList {
		a := attributes.New()
		for k, v := range serverInfo.MateData {
			a.WithValues(k, v)
		}
		address = append(address, resolver.Address{
			Addr:       serverInfo.FullAddress(),
			ServerName: s.serverName,
			Attributes: a,
		})
	}
	state.Addresses = address
	s.cc.UpdateState(state)
}

func (s *grpcResolverBuilder) update(serverInfoList []*instance.ServerInfo) {
	s.updateServerInfoList(serverInfoList)
	s.updateState()
}
