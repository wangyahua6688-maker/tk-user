package svc

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	redisx "github.com/wangyahua6688-maker/tk-common/utils/redisx/v8"
	"tk-user/internal/config"
	"tk-user/internal/platform/database"
	"tk-user/internal/repo"
	"tk-user/internal/services"
)

// ServiceContext 用户域服务上下文。
type ServiceContext struct {
	// Config 保存启动配置。
	Config config.Config
	// Redis 供仓储层缓存复用（列表缓存、评论分组缓存）。
	Redis *redis.Client
	// AuthService 用户域鉴权服务。
	AuthService *services.AuthService
	// ForumService 用户域论坛服务。
	ForumService *services.ForumService
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

	// 3) 按模块构建仓储层，注入 DB + Redis + 缓存与会话配置。
	authRepo := repo.NewAuthRepository(
		db,
		redisClient,
		c.CacheRedis.CacheTTLSeconds,
		c.UserAuth.AccessTokenTTLSeconds,
		c.UserAuth.RefreshTokenTTLSeconds,
		c.UserAuth.SMSCodeTTLSeconds,
	)
	forumRepo := repo.NewForumRepository(
		db,
		redisClient,
		c.CacheRedis.CacheTTLSeconds,
		c.UserAuth.AccessTokenTTLSeconds,
		c.UserAuth.RefreshTokenTTLSeconds,
		c.UserAuth.SMSCodeTTLSeconds,
	)
	// 4) 构建服务层，显式表达模块边界。
	authService := services.NewAuthService(authRepo)
	forumService := services.NewForumService(forumRepo)
	// 4) 返回服务上下文。
	return &ServiceContext{
		Config:       c,
		Redis:        redisClient,
		AuthService:  authService,
		ForumService: forumService,
	}, nil
}
