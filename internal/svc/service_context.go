package svc

import (
	"context"
	"fmt"
	"strings"

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

// NewServiceContext 创建ServiceContext实例。
func NewServiceContext(c config.Config) (*ServiceContext, error) {
	// 1) 初始化数据库连接。
	db, err := database.NewMySQL(c.Database.DSN)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 2) 初始化 Redis 客户端（仓储层内部按需使用）。
	redisCfg := redisx.DefaultConfig()
	// 更新当前变量或字段值。
	redisCfg.Addr = strings.TrimSpace(c.CacheRedis.Addr)
	// 更新当前变量或字段值。
	redisCfg.Password = c.CacheRedis.Password
	// 更新当前变量或字段值。
	redisCfg.DB = c.CacheRedis.DB
	// 判断条件并进入对应分支逻辑。
	if redisCfg.Addr == "" {
		// 返回当前处理结果。
		return nil, fmt.Errorf("cache redis addr is empty")
	}
	// 定义并初始化当前变量。
	redisClient, err := redisx.NewClient(context.Background(), redisCfg)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, fmt.Errorf("init redis failed: %w", err)
	}

	// 3) 构建论坛仓储层，注入 DB + Redis + 缓存 TTL。
	commentRepo := repo.NewRepository(
		// 处理当前语句逻辑。
		db,
		// 处理当前语句逻辑。
		redisClient,
		// 处理当前语句逻辑。
		c.CacheRedis.CacheTTLSeconds,
		// 处理当前语句逻辑。
		c.UserAuth.AccessTokenTTLSeconds,
		// 处理当前语句逻辑。
		c.UserAuth.RefreshTokenTTLSeconds,
		// 处理当前语句逻辑。
		c.UserAuth.SMSCodeTTLSeconds,
	)
	// 4) 返回服务上下文。
	return &ServiceContext{Config: c, Redis: redisClient, CommentRepo: commentRepo}, nil
}
