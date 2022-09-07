/**
 * @Author raven
 * @Description
 * @Date 2022/8/29
 **/
package instance

import (
	"fmt"
	"strconv"
)

const (
	etcdKeyNamePrefix = "/grpc/%s"
	etcdKeyNameFormat = etcdKeyNamePrefix + "/%s:%d"
)

type ServerInfo struct {
	Key      string                 `json:"key"`
	Name     string                 `json:"name"`
	Addr     string                 `json:"addr"`
	Port     int                    `json:"port"`
	MateData map[string]interface{} `json:"mate_data"`
}

func (info *ServerInfo) BuildPath() string {
	return fmt.Sprintf(etcdKeyNameFormat, info.Name, info.Addr, info.Port)
}

func (info *ServerInfo) FullAddress() string {
	return info.Addr + ":" + strconv.Itoa(info.Port)
}

func BuildServerPrefix(serverName string) string {
	return fmt.Sprintf(etcdKeyNamePrefix, serverName)
}
