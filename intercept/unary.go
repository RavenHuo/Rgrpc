/**
 * @Author raven
 * @Description
 * @Date 2023/4/17
 **/
package intercept

import (
	"context"
	"errors"
	"fmt"
	"github.com/RavenHuo/go-kit/log"
	"google.golang.org/grpc"
	"runtime/debug"
	"time"
)

// UnaryInterceptor 服务端拦截器，用于打印日志
func UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	startTime := time.Now()
	log.Infof(ctx, "#grpc# server start request %v` req:%+v in %.3fms, error is %v", info.FullMethod, req, err)
	defer func() {

		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("%v", r))
			log.Errorf(ctx, "grpc recover %v", string(debug.Stack()))

			resp = err
		}

		tm := float32(time.Since(startTime).Nanoseconds()) / 1000000
		log.Infof(ctx, "#grpc# server finish request %v` req:%+v,resp:%+v in %.3fms, error is %v", info.FullMethod, req, resp, tm, err)

	}()
	resp, err = handler(ctx, req)

	return resp, err
}
