package repo

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// AuthRepository 封装认证模块的数据访问依赖。
type AuthRepository struct {
	*baseRepository
}

// NewAuthRepository 创建认证模块仓储。
func NewAuthRepository(
	db *gorm.DB,
	redisClient *redis.Client,
	cacheTTLSeconds int,
	accessTokenTTLSeconds int,
	refreshTokenTTLSeconds int,
	smsCodeTTLSeconds int,
) *AuthRepository {
	return &AuthRepository{
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
