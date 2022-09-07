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
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

type GrpcResolverBuilder struct {
	serverName     string
	option         *options.GrpcOptions
	etcdClient     *etcd_client.Client
	logger         log.ILogger
	closeCh        chan struct{}
	cc             resolver.ClientConn
	serverInfoList []*instance.ServerInfo
	rwMutex        sync.RWMutex
}

// impl Resolver resolver/resolver.go:248
func (s *GrpcResolverBuilder) ResolveNow(nowOptions resolver.ResolveNowOptions) {
	s.logger.Infof(context.Background(), "ResolveNow req:%+v", nowOptions)
}

// impl Resolver resolver/resolver.go:248
func (s *GrpcResolverBuilder) Close() {
	s.closeCh <- struct{}{}
}

func newSimpleBuilder(serverName string, logger log.ILogger, option ...options.GrpcOption) (*GrpcResolverBuilder, error) {
	defaultOptions := options.DefaultRegisterOption(option...)
	return &GrpcResolverBuilder{
		serverName: serverName,
		option:     defaultOptions,
		logger:     logger,
		rwMutex:    sync.RWMutex{},
	}, nil
}

// impl Builder resolver/resolver.go:227
// grpc服务发现的主方法
func (s *GrpcResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
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
func (s *GrpcResolverBuilder) Scheme() string {
	return s.serverName
}

// serverName:需要服务发现的服务名
func MustBuildSimpleBuilder(serverName string, logger log.ILogger, option ...options.GrpcOption) *GrpcResolverBuilder {
	builder, err := newSimpleBuilder(serverName, logger, option...)
	if err != nil {
		logger.Errorf(context.Background(), "new simpleBuilder err :%s", err)
		panic("new simpleBuilder failed")
	}
	return builder
}

func (s *GrpcResolverBuilder) listen() error {
	serverIndoList, err := s.listenServerInfo()
	if err != nil {
		return err
	}
	s.updateServerInfo(serverIndoList)
	// keepAlive heartbeat and watch etcd key
	go s.keepAliveListen()
	return nil
}

func (s *GrpcResolverBuilder) listenServerInfo() ([]*instance.ServerInfo, error) {
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

func (s *GrpcResolverBuilder) keepAliveListen() () {
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
			if err == nil {
				s.updateServerInfo(serverInfoList)
			}
		// watch etcd prefix when update key
		case e := <-watchChan:
			kv := e.Kv
			var serverInfo instance.ServerInfo
			err := json.Unmarshal(kv.Value, &serverInfo)
			if err != nil {
				s.logger.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, string(kv.Key), err)
			} else {
				serverInfoList := s.listServerInfo()
				if e.Type == mvccpb.PUT {
					serverInfoList = append(serverInfoList, &serverInfo)
					s.updateServerInfo(serverInfoList)
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
						s.updateServerInfo(serverInfoList)
					}
				}
			}
		// close
		case <-s.closeCh:
			return
		}
	}
}

func (s *GrpcResolverBuilder) updateServerInfo(serverInfoList []*instance.ServerInfo) {
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	s.serverInfoList = serverInfoList
}

func (s *GrpcResolverBuilder) listServerInfo() []*instance.ServerInfo {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.serverInfoList
}

func (s *GrpcResolverBuilder) updateState() {
	state := resolver.State{}
	address := make([]resolver.Address, 0, len(s.serverInfoList))
	serverInfoList := s.listServerInfo()
	for _, serverInfo := range serverInfoList {
		address = append(address, resolver.Address{
			Addr:       serverInfo.FullAddress(),
			ServerName: s.serverName,
			Attributes: attributes.New(serverInfo.MateData),
		})
	}
	s.cc.UpdateState(state)
}
