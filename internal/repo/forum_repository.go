package repo

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// ForumRepository 封装论坛模块的数据访问依赖。
type ForumRepository struct {
	*baseRepository
}

// NewForumRepository 创建论坛模块仓储。
func NewForumRepository(
	db *gorm.DB,
	redisClient *redis.Client,
	cacheTTLSeconds int,
	accessTokenTTLSeconds int,
	refreshTokenTTLSeconds int,
	smsCodeTTLSeconds int,
) *ForumRepository {
	return &ForumRepository{
		baseRepository: newBaseRepository(
			db,
			redisClient,
			cacheTTLSeconds,
			accessTokenTTLSeconds,
			refreshTokenTTLSeconds,
			smsCodeTTLSeconds,
		),
	}
}
