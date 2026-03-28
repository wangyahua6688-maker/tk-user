package repo

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	redisx "github.com/wangyahua6688-maker/tk-common/utils/redisx/v8"
	"gorm.io/gorm"
)

// baseRepository 提供仓储层共享的底层依赖与通用缓存能力。
type baseRepository struct {
	// 处理当前语句逻辑。
	db *gorm.DB
	// 处理当前语句逻辑。
	redis *redis.Client
	// 处理当前语句逻辑。
	cacheTTL time.Duration
	// accessTokenTTL 控制访问令牌过期时间。
	accessTokenTTL time.Duration
	// refreshTokenTTL 控制刷新令牌过期时间。
	refreshTokenTTL time.Duration
	// smsCodeTTL 控制短信验证码有效期。
	smsCodeTTL time.Duration
}

// newBaseRepository 创建共享仓储基础设施实例。
func newBaseRepository(
	// 处理当前语句逻辑。
	db *gorm.DB,
	// 处理当前语句逻辑。
	redisClient *redis.Client,
	// 处理当前语句逻辑。
	cacheTTLSeconds int,
	// 处理当前语句逻辑。
	accessTokenTTLSeconds int,
	// 处理当前语句逻辑。
	refreshTokenTTLSeconds int,
	// 处理当前语句逻辑。
	smsCodeTTLSeconds int,
	// 进入新的代码块进行处理。
) *baseRepository {
	// 默认缓存 10 秒，兼顾及时性与数据库压力。
	ttl := 10 * time.Second
	// 判断条件并进入对应分支逻辑。
	if cacheTTLSeconds > 0 {
		// 更新当前变量或字段值。
		ttl = time.Duration(cacheTTLSeconds) * time.Second
	}
	// 定义并初始化当前变量。
	accessTTL := 24 * time.Hour
	// 判断条件并进入对应分支逻辑。
	if accessTokenTTLSeconds > 0 {
		// 更新当前变量或字段值。
		accessTTL = time.Duration(accessTokenTTLSeconds) * time.Second
	}
	// 定义并初始化当前变量。
	refreshTTL := 7 * 24 * time.Hour
	// 判断条件并进入对应分支逻辑。
	if refreshTokenTTLSeconds > 0 {
		// 更新当前变量或字段值。
		refreshTTL = time.Duration(refreshTokenTTLSeconds) * time.Second
	}
	// 定义并初始化当前变量。
	codeTTL := 5 * time.Minute
	// 判断条件并进入对应分支逻辑。
	if smsCodeTTLSeconds > 0 {
		// 更新当前变量或字段值。
		codeTTL = time.Duration(smsCodeTTLSeconds) * time.Second
	}
	// 返回当前处理结果。
	return &baseRepository{
		// 处理当前语句逻辑。
		db: db,
		// 处理当前语句逻辑。
		redis: redisClient,
		// 处理当前语句逻辑。
		cacheTTL: ttl,
		// 处理当前语句逻辑。
		accessTokenTTL: accessTTL,
		// 处理当前语句逻辑。
		refreshTokenTTL: refreshTTL,
		// 处理当前语句逻辑。
		smsCodeTTL: codeTTL,
	}
}

// loadCache 从 Redis 读取缓存并反序列化。
func (r *baseRepository) loadCache(ctx context.Context, key string, out any) bool {
	// 统一复用 common-utils 的 Redis JSON 读取逻辑。
	hit, err := redisx.GetJSON(ctx, r.redis, key, out)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return false
	}
	// 返回当前处理结果。
	return hit
}

// saveCache 将对象序列化后写入 Redis。
func (r *baseRepository) saveCache(ctx context.Context, key string, data any) {
	// 无 Redis 或 TTL 非法时不写缓存。
	if r.redis == nil || r.cacheTTL <= 0 {
		// 返回当前处理结果。
		return
	}
	// 写缓存失败不抛错，避免影响主流程。
	_ = redisx.SetJSON(ctx, r.redis, key, data, r.cacheTTL)
}
