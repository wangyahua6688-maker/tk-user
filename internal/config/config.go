package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 用户域服务配置（当前承载论坛读接口）。
type Config struct {
	zrpc.RpcServerConf // 用户域 gRPC 服务监听配置。

	Database struct { // 数据库分组。
		DSN string // 用户域业务库 DSN 连接串。
	} // 数据库配置。

	CacheRedis struct { // 缓存分组。
		Addr            string // Redis 地址。
		Password        string // Redis 密码。
		DB              int    // Redis 分库编号。
		CacheTTLSeconds int    // 评论聚合缓存 TTL（秒）。
	} // 缓存配置。

	UserAuth struct { // 用户认证分组。
		AccessTokenTTLSeconds  int // 访问令牌有效期（秒）。
		RefreshTokenTTLSeconds int // 刷新令牌有效期（秒）。
		SMSCodeTTLSeconds      int // 短信验证码有效期（秒）。
	} // 用户认证配置。
}
