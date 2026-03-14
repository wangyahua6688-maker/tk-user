package svc

import (
	"context"

	"github.com/go-redis/redis/v8"
	redisx "tk-common/utils/redisx/v8"
	"tk-user/internal/config"
	"tk-user/internal/platform/database"
	"tk-user/internal/repo"
)

// ServiceContext 用户域服务上下文。
type ServiceContext struct {
	// Config 保存启动配置。
	Config config.Config
	// Redis 供仓储层缓存复用（列表缓存、评论分组缓存）。
	Redis *redis.Client
	// CommentRepo 用户域论坛/评论仓储。
	CommentRepo *repo.Repository
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	// 1) 初始化数据库连接。
	db, err := database.NewMySQL(c.Database.DSN)
	if err != nil {
		return nil, err
	}

	// 2) 初始化 Redis 客户端（仓储层内部按需使用）。
	redisCfg := redisx.DefaultConfig()
	redisCfg.Addr = c.CacheRedis.Addr
	redisCfg.Password = c.CacheRedis.Password
	redisCfg.DB = c.CacheRedis.DB
	redisClient, _ := redisx.NewClient(context.Background(), redisCfg)

	// 3) 构建论坛仓储层，注入 DB + Redis + 缓存 TTL。
	commentRepo := repo.NewRepository(
		db,
		redisClient,
		c.CacheRedis.CacheTTLSeconds,
		c.Auth.AccessTokenTTLSeconds,
		c.Auth.RefreshTokenTTLSeconds,
		c.Auth.SMSCodeTTLSeconds,
	)
	// 4) 返回服务上下文。
	return &ServiceContext{Config: c, Redis: redisClient, CommentRepo: commentRepo}, nil
}
