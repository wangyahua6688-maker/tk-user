package main

import (
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	tkv1 "tk-proto/tk/v1"
	"tk-user/internal/config"
	"tk-user/internal/server"
	"tk-user/internal/svc"
)

func main() {
	// 默认读取用户域配置文件。
	var configFile = flag.String("f", "etc/user.yaml", "the config file")
	flag.Parse()

	var c config.Config
	// 加载 RPC/数据库/缓存配置。
	conf.MustLoad(*configFile, &c)

	// 初始化服务上下文（DB + Redis + Repository）。
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(fmt.Sprintf("init tk-user failed: %v", err))
	}

	// 注册用户域 gRPC 服务实现。
	rpcServer := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		tkv1.RegisterUserServiceServer(grpcServer, server.NewUserServer(svcCtx))
	})
	defer rpcServer.Stop()

	// 输出启动日志，便于排查端口/环境问题。
	logx.Infof("starting tk-user rpc on %s", c.ListenOn)
	// 进入阻塞监听循环。
	rpcServer.Start()
}
