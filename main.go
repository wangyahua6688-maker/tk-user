package main

import (
	"flag"
	"fmt"

	tkv1 "github.com/wangyahua6688-maker/tk-proto/gen/go/tk/v1"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"tk-user/internal/config"
	"tk-user/internal/server"
	"tk-user/internal/svc"
)

// main 启动程序入口。
func main() {
	// 默认读取用户域配置文件。
	var configFile = flag.String("f", "etc/user.yaml", "the config file")
	// 调用flag.Parse完成当前处理。
	flag.Parse()

	// 声明当前变量。
	var c config.Config
	// 加载 RPC/数据库/缓存配置。
	conf.MustLoad(*configFile, &c)

	// 初始化服务上下文（DB + Redis + Repository）。
	svcCtx, err := svc.NewServiceContext(c)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 调用panic完成当前处理。
		panic(fmt.Sprintf("init tk-user failed: %v", err))
	}

	// 注册用户域 gRPC 服务实现。
	rpcServer := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		// 调用tkv1.RegisterUserServiceServer完成当前处理。
		tkv1.RegisterUserServiceServer(grpcServer, server.NewUserServer(svcCtx))
	})
	// 注册延迟执行逻辑。
	defer rpcServer.Stop()

	// 输出启动日志，便于排查端口/环境问题。
	logx.Infof("starting tk-user rpc on %s", c.ListenOn)
	// 进入阻塞监听循环。
	rpcServer.Start()
}
