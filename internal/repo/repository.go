package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
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
	// Redis 未配置时直接跳过缓存路径。
	if r.redis == nil {
		return false
	}
	// 读取缓存原始字节。
	raw, err := r.redis.Get(ctx, key).Bytes()
	if err != nil || len(raw) == 0 {
		return false
	}
	// 反序列化失败视为缓存无效。
	return json.Unmarshal(raw, out) == nil
}

// saveCache 将对象序列化后写入 Redis。
func (r *Repository) saveCache(ctx context.Context, key string, data any) {
	// 无 Redis 或 TTL 非法时不写缓存。
	if r.redis == nil || r.cacheTTL <= 0 {
		return
	}
	// 序列化失败直接丢弃，不影响主业务。
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	// 写缓存失败不抛错，避免影响主流程。
	_ = r.redis.Set(ctx, key, raw, r.cacheTTL).Err()
}
