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
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/options"
	"github.com/coreos/etcd/clientv3"
	"os"
	"time"
)

// 服务注册
type Register struct {
	serverInfo *instance.ServerInfo
	option     *options.GrpcOptions
	leasesID   clientv3.LeaseID
	logger     log.ILogger
	etcdClient *etcd_client.Client
	closeCh    chan struct{}
}

func NewRegister(logger log.ILogger, opts ...options.GrpcOption) (*Register, error) {
	registerOptions := options.DefaultRegisterOption(opts...)
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: registerOptions.Endpoints()}, logger)
	if err != nil {
		logger.Errorf(context.Background(), "grpc register server init etcd pb endpoints:%+v, err:%s", registerOptions.Endpoints(), err)
		return nil, err
	}
	return &Register{
		option:     registerOptions,
		logger:     logger,
		etcdClient: etcdClient,
		closeCh:    make(chan struct{}),
	}, nil
}

func (r *Register) Register(info *instance.ServerInfo) error {

	if info.Addr == "" {
		info.Addr = getLocalIpAddr()
	}
	if info.Addr == "" || info.Port == 0 {
		return errors.New("register info must not empty")
	}
	r.serverInfo = info
	err := r.register(info)
	if err != nil {
		return err
	}
	r.logger.Infof(context.Background(), "register server success  info :%+v ", info)
	go r.keepAlive()

	return nil
}

func (r *Register) Unregister() error {
	if r.etcdClient == nil || r.serverInfo == nil {
		return nil
	}
	r.closeCh <- struct{}{}
	if err := r.etcdClient.DeleteKey(r.serverInfo.BuildPath()); err != nil {
		r.logger.Errorf(context.Background(), "unregister server failed info:%+v, err:%s", r.serverInfo, err)
		return err
	}
	return nil
}

func (r *Register) register(info *instance.ServerInfo) error {
	info.Key = info.BuildPath()
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
