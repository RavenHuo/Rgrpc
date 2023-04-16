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
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"sync"
	"time"
)

// 自己实现的服务发现
type Discovery struct {
	option     *options.GrpcOptions
	serversMap map[string][]*instance.ServerInfo
	rwMutex    sync.RWMutex
	closeCh    chan struct{}
	etcdClient *etcd_client.Client
}

func NewDiscovery(option ...options.GrpcOption) (*Discovery, error) {
	defaultOptions := options.DefaultRegisterOption(option...)
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: defaultOptions.Endpoints()})
	if err != nil {
		log.Errorf(context.Background(), "grpc register server init etcd pb endpoints:%+v, err:%s", defaultOptions.Endpoints(), err)
		return nil, err
	}
	return &Discovery{
		option:     defaultOptions,
		serversMap: make(map[string][]*instance.ServerInfo, 0),
		rwMutex:    sync.RWMutex{},
		etcdClient: etcdClient,
		closeCh:    make(chan struct{}),
	}, nil
}

func (d *Discovery) Listen(serverName string) error {
	if serverName == "" {
		return errors.New("listenServerInfo failed serverName must not empty")
	}
	d.rwMutex.Lock()
	if _, ok := d.serversMap[serverName]; ok {
		d.rwMutex.Unlock()
		return errors.New(fmt.Sprintf("server %s has been listenServerInfo", serverName))
	}
	d.rwMutex.Unlock()

	d.serversMap[serverName] = make([]*instance.ServerInfo, 0)
	serverInfoList, err := d.listenServerInfo(serverName)
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
	prefix := instance.GetServerPrefix(serverName)
	watchChan, err := d.etcdClient.WatchPrefix(context.Background(), prefix)
	if err != nil {
		log.Errorf(context.Background(), "watch %s err:%s", prefix, err)
	}

	for {
		select {
		// timer update serverMap
		case <-timer.C:
			serverInfoList, err := d.listenServerInfo(serverName)
			if err == nil {
				d.updateServerInfo(serverName, serverInfoList)
			}
		// watch etcd prefix when update key
		case e := <-watchChan:
			kv := e.Kv
			serverInfoList := d.ListServerInfo(serverName)

			if e.Type == mvccpb.PUT {
				var serverInfo instance.ServerInfo
				err := json.Unmarshal(kv.Value, &serverInfo)
				if err != nil {
					log.Errorf(context.Background(), "listenServerInfo %s Unmarshal %+v failed err:%s", prefix, string(kv.Key), err)
				}
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

		case <-d.closeCh:
			return
		}
	}
}

func (d *Discovery) listenServerInfo(serverName string) ([]*instance.ServerInfo, error) {
	prefix := instance.GetServerPrefix(serverName)
	resp, err := d.etcdClient.GetDirectory(context.Background(), prefix)
	if err != nil {
		return nil, err
	}
	result := make([]*instance.ServerInfo, 0, len(resp))
	for _, v := range resp {
		var serverInfo instance.ServerInfo
		err := json.Unmarshal(v, &serverInfo)
		if err != nil {
			log.Errorf(context.Background(), "listenServerInfo %s Unmarshal %+v failed err:%s", prefix, v, err)
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
