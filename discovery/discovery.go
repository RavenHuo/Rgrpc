/**
 * @Author raven
 * @Description
 * @Date 2022/8/31
 **/
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/options"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"sync"
	"time"
)

type Discovery struct {
	option     *options.GrpcOptions
	serversMap map[string][]*instance.ServerInfo
	rwMutex    sync.RWMutex
	closeCh    chan struct{}
	etcdClient *etcd_client.Client
	logger     log.ILogger
}

func NewDiscovery(option *options.GrpcOptions, logger log.ILogger) (*Discovery, error) {
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: option.Endpoints()}, logger)
	if err != nil {
		logger.Errorf(context.Background(), "grpc register server init etcd client endpoints:%+v, err:%s", option.Endpoints(), err)
		return nil, err
	}
	return &Discovery{
		option:     option,
		serversMap: make(map[string][]*instance.ServerInfo, 0),
		rwMutex:    sync.RWMutex{},
		logger:     logger,
		etcdClient: etcdClient,
		closeCh:    make(chan struct{}),
	}, nil
}

func (d *Discovery) Listen(serverName string) error {
	d.rwMutex.Lock()
	if d.etcdClient == nil {
	}
	if _, ok := d.serversMap[serverName]; ok {
		d.rwMutex.Unlock()
		return errors.New(fmt.Sprintf("server %s has been listen", serverName))
	}
	d.rwMutex.Unlock()

	d.serversMap[serverName] = make([]*instance.ServerInfo, 0)
	serverInfoList, err := d.listen(serverName)
	if err != nil {
		return err
	}
	d.updateServerInfo(serverName, serverInfoList)
	go d.keepAliveListen(serverName)
	return nil
}

func (d *Discovery) updateServerInfo(serverName string, serverInfo []*instance.ServerInfo) {
	d.rwMutex.Lock()
	defer d.rwMutex.Unlock()
	d.serversMap[serverName] = serverInfo
}

func (d *Discovery) keepAliveListen(serverName string) {
	timer := time.NewTimer(time.Duration(d.option.KeepAliveTtl()) * time.Second)
	prefix := instance.BuildServerPrefix(serverName)
	watchChan, err := d.etcdClient.WatchPrefix(context.Background(), prefix)
	if err != nil {
		d.logger.Errorf(context.Background(), "watch %s err:%s", prefix, err)
	}

	for {
		select {
		// timer update serverMap
		case <-timer.C:
			serverInfoList, err := d.listen(serverName)
			if err == nil {
				d.updateServerInfo(serverName, serverInfoList)
			}
		// watch etcd prefix when update key
		case e := <-watchChan:
			kv := e.Kv
			var serverInfo instance.ServerInfo
			err := json.Unmarshal(kv.Value, &serverInfo)
			if err != nil {
				d.logger.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, string(kv.Key), err)
			} else {
				serverInfoList := d.ListServerInfo(serverName)
				if e.Type == mvccpb.PUT {
					serverInfoList = append(serverInfoList, &serverInfo)
					d.updateServerInfo(serverName, serverInfoList)
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
						d.updateServerInfo(serverName, serverInfoList)
					}
				}
			}
		}
	}
}

func (d *Discovery) listen(serverName string) ([]*instance.ServerInfo, error) {
	prefix := instance.BuildServerPrefix(serverName)
	resp, err := d.etcdClient.GetDirectory(context.Background(), prefix)
	if err != nil {
		return nil, err
	}
	result := make([]*instance.ServerInfo, len(resp))
	for _, v := range resp {
		var serverInfo instance.ServerInfo
		err := json.Unmarshal(v, &serverInfo)
		if err != nil {
			d.logger.Errorf(context.Background(), "listen %s Unmarshal %+v failed err:%s", prefix, v, err)
		} else {
			result = append(result, &serverInfo)
		}
	}
	return result, nil
}

func (d *Discovery) ListServerInfo(serverName string) []*instance.ServerInfo {
	d.rwMutex.RLock()
	defer d.rwMutex.RUnlock()
	return d.serversMap[serverName]
}
