/**
 * @Author raven
 * @Description
 * @Date 2022/8/29
 **/
package register

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/go-kit/utils/nets"
	grpc2 "github.com/RavenHuo/grpc"
	"github.com/RavenHuo/grpc/options"
	"github.com/coreos/etcd/clientv3"
	"os"
	"time"
)

// 服务注册
type Register struct {
	serverInfo  *grpc2.ServerInfo
	option      *options.RegisterOptions
	closeCh     chan struct{}
	leasesID    clientv3.LeaseID
	logger      log.ILogger
	etcdClient  *etcd_client.Client
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse
}

func NewRegister(registerOptions *options.RegisterOptions, logger log.ILogger) *Register {
	if registerOptions == nil || logger == nil {
		panic("NewRegister failed option or serverInfo is empty")
	}
	return &Register{
		option: registerOptions,
		logger: logger,
	}
}

func (r *Register) Register(info *grpc2.ServerInfo) error {
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: r.option.Endpoints()}, r.logger)
	if err != nil {
		r.logger.Errorf(context.Background(), "grpc register server init etcd client endpoints:%+v, err:%s", r.option.Endpoints(), err)
		return err
	}
	if info.Addr == "" {
		info.Addr = getLocalIpAddr()
	}
	if info.Addr == "" || info.Port == 0 {
		return errors.New("")
	}
	r.serverInfo = info
	r.etcdClient = etcdClient
	err = r.register(info)
	if err != nil {
		return err
	}
	go r.keepAlive()

	return nil
}

func (r *Register) Unregister() error {
	if r.etcdClient == nil {
		return nil
	}
	r.closeCh <- struct{}{}
	if err := r.etcdClient.DeleteKey(r.serverInfo.BuildPath()); err != nil {
		r.logger.Errorf(context.Background(), "unregister server failed info:%+v, err:%s", r.serverInfo, err)
		return err
	}
	return nil
}

func (r *Register) register(info *grpc2.ServerInfo) error {
	jsonInfo, _ := json.Marshal(info)
	if _, err := r.etcdClient.PutKey(info.BuildPath(), string(jsonInfo), r.option.KeepAliveTtl()); err != nil {
		r.logger.Errorf(context.Background(), "server register failed serverInfo:%+v, err:%s", info, err)
		return err
	}
	return nil
}

func (r *Register) keepAlive() {
	timer := time.NewTimer(time.Duration(r.option.KeepAliveTtl()) * time.Second)
	for {
		select {
		case <-timer.C:
			r.logger.Infof(context.Background(), "register keep alive info:%+v", r.serverInfo)
			r.register(r.serverInfo)
		case <-r.closeCh:
			return
		}
	}
}

func getLocalIpAddr() string {
	addr, _, err := nets.GetLocalMainIP()
	if err != nil {
		hostname, _ := os.Hostname()
		return hostname
	}
	return addr
}
