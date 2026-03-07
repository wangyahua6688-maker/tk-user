package config

import "github.com/zeromicro/go-zero/zrpc"

// Config 用户域服务配置（当前承载论坛读接口）。
type Config struct {
	// RpcServerConf：用户域 gRPC 服务监听配置。
	zrpc.RpcServerConf
	// Database：用户域业务库连接配置。
	Database struct {
		DSN string
	}
	// CacheRedis：论坛列表/评论分组缓存配置。
	CacheRedis struct {
		// Addr Redis 地址。
		Addr string
		// Password Redis 密码。
		Password string
		// DB Redis 分库编号。
		DB int
		// CacheTTLSeconds 评论聚合缓存 TTL（秒）。
		CacheTTLSeconds int
	}
	// Auth 用户认证与短信登录配置。
	Auth struct {
		// AccessTokenTTLSeconds 访问令牌有效期（秒）。
		AccessTokenTTLSeconds int
		// RefreshTokenTTLSeconds 刷新令牌有效期（秒）。
		RefreshTokenTTLSeconds int
		// SMSCodeTTLSeconds 验证码有效期（秒）。
		SMSCodeTTLSeconds int
	}
}
