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
	"github.com/RavenHuo/Rgrpc/instance"
	"github.com/RavenHuo/Rgrpc/options"
	"github.com/RavenHuo/go-kit/etcd_client"
	"github.com/RavenHuo/go-kit/log"
	"github.com/RavenHuo/go-kit/utils/nets"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
)

// 服务注册
type Register struct {
	serverInfo *instance.ServerInfo
	option     *options.GrpcOptions
	etcdClient *etcd_client.Client
	closeCh    chan struct{}
}

func NewRegister(opts ...options.GrpcOption) (*Register, error) {
	registerOptions := options.DefaultRegisterOption(opts...)
	etcdClient, err := etcd_client.New(&etcd_client.EtcdConfig{Endpoints: registerOptions.Endpoints()})
	if err != nil {
		log.Errorf(context.Background(), "grpc register server init etcd pb endpoints:%+v, err:%s", registerOptions.Endpoints(), err)
		return nil, err
	}
	return &Register{
		option:     registerOptions,
		etcdClient: etcdClient,
		closeCh:    make(chan struct{}),
	}, nil
}

func (r *Register) Register(info *instance.ServerInfo) error {

	if info.Ip == "" {
		info.Ip = getLocalIpAddr()
	}
	if info.Ip == "" || info.Port == 0 {
		return errors.New("register info must not empty")
	}
	r.serverInfo = info
	leaseId, err := r.register(info)
	if err != nil {
		return err
	}
	log.Infof(context.Background(), "register server success  info :%+v ", info)
	go r.keepAlive(leaseId)

	return nil
}

func (r *Register) Unregister() error {
	if r.etcdClient == nil || r.serverInfo == nil {
		return nil
	}
	r.closeCh <- struct{}{}
	if err := r.etcdClient.DeleteKey(r.serverInfo.BuildPath()); err != nil {
		log.Errorf(context.Background(), "unregister server failed info:%+v, err:%s", r.serverInfo, err)
		return err
	}
	log.Infof(context.Background(), "unregister success serverInfo:%+v", r.serverInfo)
	return nil
}

func (r *Register) register(info *instance.ServerInfo) (int64, error) {
	info.Key = info.BuildPath()
	jsonInfo, _ := json.Marshal(info)
	leaseId, err := r.etcdClient.PutKey(info.BuildPath(), string(jsonInfo), r.option.LeaseTimestamp())
	if err != nil {
		log.Errorf(context.Background(), "server register failed serverInfo:%+v, err:%s", info, err)
		return 0, err
	}
	return leaseId, nil
}

func (r *Register) keepAlive(leaseId int64) {
	keepAliveCh, err := r.etcdClient.GetClient().KeepAlive(context.Background(), clientv3.LeaseID(leaseId))
	if err != nil {
		return
	}
	for {
		select {
		case <-keepAliveCh:
			log.Infof(context.Background(), "register keep alive info:%+v", r.serverInfo)
		case <-r.closeCh:
			log.Infof(context.Background(), "register listen close channel info:%+v", r.serverInfo)
			return
		}
	}
}

func getLocalIpAddr() string {
	ip, _, err := nets.GetLocalMainIP()
	if err != nil {
		hostname, _ := os.Hostname()
		return hostname
	}
	return ip
}
