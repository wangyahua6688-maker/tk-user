package repo

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	redisx "tk-common/utils/redisx/v8"
)

// Repository 评论微服务数据访问层。
// 侧重点：
// - 高频读（列表、分组）；
// - 短期缓存（Redis）；
// - 单次聚合查询，减少 N+1 开销。
type Repository struct {
	db       *gorm.DB
	redis    *redis.Client
	cacheTTL time.Duration
	// accessTokenTTL 控制访问令牌过期时间。
	accessTokenTTL time.Duration
	// refreshTokenTTL 控制刷新令牌过期时间。
	refreshTokenTTL time.Duration
	// smsCodeTTL 控制短信验证码有效期。
	smsCodeTTL time.Duration
}

// NewRepository 创建论坛仓储层实例。
func NewRepository(
	db *gorm.DB,
	redisClient *redis.Client,
	cacheTTLSeconds int,
	accessTokenTTLSeconds int,
	refreshTokenTTLSeconds int,
	smsCodeTTLSeconds int,
) *Repository {
	// 默认缓存 10 秒，兼顾及时性与数据库压力。
	ttl := 10 * time.Second
	if cacheTTLSeconds > 0 {
		ttl = time.Duration(cacheTTLSeconds) * time.Second
	}
	accessTTL := 24 * time.Hour
	if accessTokenTTLSeconds > 0 {
		accessTTL = time.Duration(accessTokenTTLSeconds) * time.Second
	}
	refreshTTL := 7 * 24 * time.Hour
	if refreshTokenTTLSeconds > 0 {
		refreshTTL = time.Duration(refreshTokenTTLSeconds) * time.Second
	}
	codeTTL := 5 * time.Minute
	if smsCodeTTLSeconds > 0 {
		codeTTL = time.Duration(smsCodeTTLSeconds) * time.Second
	}
	return &Repository{
		db:              db,
		redis:           redisClient,
		cacheTTL:        ttl,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		smsCodeTTL:      codeTTL,
	}
}

// loadCache 从 Redis 读取缓存并反序列化。
func (r *Repository) loadCache(ctx context.Context, key string, out any) bool {
	// 统一复用 common-utils 的 Redis JSON 读取逻辑。
	hit, err := redisx.GetJSON(ctx, r.redis, key, out)
	if err != nil {
		return false
	}
	return hit
}

// saveCache 将对象序列化后写入 Redis。
func (r *Repository) saveCache(ctx context.Context, key string, data any) {
	// 无 Redis 或 TTL 非法时不写缓存。
	if r.redis == nil || r.cacheTTL <= 0 {
		return
	}
	// 写缓存失败不抛错，避免影响主流程。
	_ = redisx.SetJSON(ctx, r.redis, key, data, r.cacheTTL)
}
