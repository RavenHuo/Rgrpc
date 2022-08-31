/**
 * @Author raven
 * @Description
 * @Date 2022/8/29
 **/
package grpc

import "fmt"

const (
	EtcdKeyNameFormat = "/grpc/%s/%s:%d"
)

type ServerInfo struct {
	Name     string
	Addr     string
	Port     int
	MateData map[string]interface{}
}

func (info *ServerInfo) BuildPath() string {
	return fmt.Sprintf(EtcdKeyNameFormat, info.Name, info.Addr, info.Port)
}
