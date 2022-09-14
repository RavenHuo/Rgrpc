/**
 * @Author raven
 * @Description
 * @Date 2022/9/13
 **/
package version_weight

import (
	"encoding/json"
	"errors"
	"github.com/RavenHuo/grpc/instance"
	"github.com/RavenHuo/grpc/utils"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"sync"
)

const Name = "version_weight"

var (
	ErrNoAvailableConn    = errors.New("no available conn")
	ErrNoMatchVersionConn = errors.New("no match version conn")
)

// versionWeightPickerBuilder to create picker impl balancer/base/base.go:39
// 通过 versionWeightPickerBuilder.Build() 方法构建选择器
type versionWeightPickerBuilder struct {
}

func (g versionWeightPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	// ReadySCs 是从所有准备好的 SubConns 到用于创建它们的地址的映射。
	// 没有可用的地址
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var balancerConnServerInfos = make([]*balancerConnServerInfo, 0, len(info.ReadySCs))
	for conn, addr := range info.ReadySCs {
		serverInfo := instance.BuilderServerInfo(addr.Address)
		balancerConnServerInfos = append(balancerConnServerInfos,
			&balancerConnServerInfo{
				subConn:    conn,
				serverInfo: serverInfo,
			})
	}
	if len(balancerConnServerInfos) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	return &versionWeightPicker{
		balancerConnServerInfos: balancerConnServerInfos,
	}
}

type versionWeightPicker struct {
	balancerConnServerInfos []*balancerConnServerInfo
	mu                      sync.Mutex
	// 用于轮询的下标
	next int
}

// Pick PickInfo 这里面包含grpc的Context
// Pick 在此方法选择一个可用的链接
func (p *versionWeightPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {

	if len(p.balancerConnServerInfos) == 0 {
		return balancer.PickResult{}, ErrNoAvailableConn
	}
	ctxVersion := utils.GetGrpcHeader(info.Ctx, instance.VersionWeightHeader)
	var ctxVersionMap map[string]float64
	if ctxVersion != "" {
		_ = json.Unmarshal([]byte(ctxVersion), &ctxVersionMap)
	}
	// 没有配置版本权重,使用轮询的方法
	if ctxVersion == "" || len(ctxVersionMap) == 0 {
		p.mu.Lock()
		subConns := make([]balancer.SubConn, 0, len(p.balancerConnServerInfos))
		for _, connServerInfo := range p.balancerConnServerInfos {
			subConns = append(subConns, connServerInfo.subConn)
		}
		p.next = (p.next + 1) % len(p.balancerConnServerInfos)
		sc := subConns[p.next]
		p.mu.Unlock()
		return balancer.PickResult{SubConn: sc}, nil
	}
	connWeightMap := make(map[interface{}]float64, 0)

	// filter version
	for _, connServerInfo := range p.balancerConnServerInfos {
		version := connServerInfo.serverInfo.Version()
		if weight, ok := ctxVersionMap[version]; ok {
			connWeightMap[connServerInfo.subConn] = weight
		}
	}

	if len(connWeightMap) == 0 {
		return balancer.PickResult{}, ErrNoMatchVersionConn
	}
	randomWeightMap := utils.NewWeightRandomMap(connWeightMap)
	randomResult := randomWeightMap.Random().(balancer.SubConn)

	return balancer.PickResult{SubConn: randomResult}, nil
}

type balancerConnServerInfo struct {
	subConn    balancer.SubConn
	serverInfo *instance.ServerInfo
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, &versionWeightPickerBuilder{}, base.Config{HealthCheck: true})
}
func init() {
	balancer.Register(newBuilder())
}
