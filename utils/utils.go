/**
 * @Author raven
 * @Description
 * @Date 2022/9/13
 **/
package utils

import (
	"context"
	"google.golang.org/grpc/metadata"
)

func GetGrpcHeader(ctx context.Context, name string) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if md != nil {
		valueArr := md.Get(name)
		if valueArr != nil && len(valueArr) > 0 {
			return valueArr[0]
		}
	}
	return ""
}
