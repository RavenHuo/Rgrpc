/**
 * @Author raven
 * @Description
 * @Date 2022/8/29
 **/
package instance

import (
	"fmt"
	"github.com/RavenHuo/go-kit/utils/nets"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"strconv"
)

const (
	etcdKeyNamePrefix = "/grpc/%s"
	etcdKeyNameFormat = etcdKeyNamePrefix + "/%s:%d"
	// VersionWeightHeader 版本权重 header key
	VersionWeightHeader = "version_weight"
	Version             = "version"
	serverInfoKey       = "server_info"
)

type ServerInfo struct {
	Key      string                 `json:"key"`
	Name     string                 `json:"name"`
	Ip       string                 `json:"ip"`
	Port     int                    `json:"port"`
	MateData map[string]interface{} `json:"mate_data"`
}

func (info *ServerInfo) BuildPath() string {
	return fmt.Sprintf(etcdKeyNameFormat, info.Name, info.Ip, info.Port)
}

func (info *ServerInfo) FullAddress() string {
	return info.Ip + ":" + strconv.Itoa(info.Port)
}

// 注册的版本
func (info *ServerInfo) Version() string {
	if version, ok := info.MateData[Version]; ok {
		if versionStr, ok2 := version.(string); ok2 {
			return versionStr
		}
	}
	return ""
}

func GetServerPrefix(serverName string) string {
	return fmt.Sprintf(etcdKeyNamePrefix, serverName)
}

func BuilderAddress(serverInfo *ServerInfo) resolver.Address {
	a := attributes.New().WithValues(serverInfoKey, serverInfo.MateData)
	addr := resolver.Address{
		Addr:       serverInfo.FullAddress(),
		ServerName: serverInfo.Name,
		Attributes: a,
	}
	return addr
}

func BuilderServerInfo(address resolver.Address) *ServerInfo {
	ip, port := nets.GetAddrAndPort(address.Addr)
	serverInfo := &ServerInfo{
		Ip:   ip,
		Port: port,
		Name: address.ServerName,
	}
	if mateData := address.Attributes.Value(serverInfoKey); mateData != nil {
		if mateDataMap, ok := mateData.(map[string]interface{}); ok {
			serverInfo.MateData = mateDataMap
		}
	}
	return serverInfo
}
