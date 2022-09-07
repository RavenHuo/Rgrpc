/**
 * @Author raven
 * @Description
 * @Date 2022/9/7
 **/
package grpc

import (
	"testing"
	"time"
)

func TestResolver(t *testing.T) {
	// etcd中注册5个服务
	go newServer(serverName, 1000)
	go newServer(serverName, 1001)
	go newServer(serverName, 1002)
	go newServer(serverName, 1003)
	go newServer(serverName, 1004)
	go newServer(serverName, 1005)
	time.Sleep(2 * time.Second)
}