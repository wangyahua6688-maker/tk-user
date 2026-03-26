// Package rediskey 定义 tk-user 服务使用的全部 Redis key 模板。
//
// 命名规范：
//   - 格式：tk:{service}:{domain}:{identifier}
//   - 分隔符：冒号（:）
//   - 标识符中的动态部分用占位符说明，通过 fmt.Sprintf 拼接
//   - 禁止在 key 中使用用户可控的原始字符串（如 nickname），必须先哈希或截断
//
// 所有 key 及其 TTL 在此文件统一维护，方便追踪和修改。
package rediskey

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────
// Session / Token
// ─────────────────────────────────────────────

const (
	// AccessTokenTTL AccessToken 默认有效期（2 小时）
	AccessTokenTTL = 2 * time.Hour
	// RefreshTokenTTL RefreshToken 默认有效期（7 天）
	RefreshTokenTTL = 7 * 24 * time.Hour
)

// KeyAccessToken 用户 AccessToken → UserID 映射。
// 格式: tk:user:access:{token}
// TTL:  AccessTokenTTL
func KeyAccessToken(token string) string {
	return fmt.Sprintf("tk:user:access:%s", token)
}

// KeyRefreshToken 用户 RefreshToken → UserID 映射。
// 格式: tk:user:refresh:{token}
// TTL:  RefreshTokenTTL
func KeyRefreshToken(token string) string {
	return fmt.Sprintf("tk:user:refresh:%s", token)
}

// KeyAccessRefreshPair AccessToken → RefreshToken 关联（便于登出时联动失效）。
// 格式: tk:user:access-refresh:{accessToken}
// TTL:  AccessTokenTTL
func KeyAccessRefreshPair(accessToken string) string {
	return fmt.Sprintf("tk:user:access-refresh:%s", accessToken)
}

// KeyUserTokenVersion 用户 Token 版本号（修改密码/封禁时递增，使旧 Token 失效）。
// 格式: tk:user:token:version:{userID}
// TTL:  永久（不设过期，由业务主动维护）
func KeyUserTokenVersion(userID uint64) string {
	return fmt.Sprintf("tk:user:token:version:%d", userID)
}

// ─────────────────────────────────────────────
// SMS 验证码
// ─────────────────────────────────────────────

const (
	// SMSCodeTTL 验证码有效期（5 分钟）
	SMSCodeTTL = 5 * time.Minute
)

// KeySMSCode 短信验证码存储。
// 格式: tk:user:sms:{purpose}:{phone}
// TTL:  SMSCodeTTL（验证成功后立即 DEL，防重放）
// purpose 取值: "login" | "register" | "reset"
func KeySMSCode(purpose, phone string) string {
	return fmt.Sprintf("tk:user:sms:%s:%s", purpose, phone)
}

// ─────────────────────────────────────────────
// SMS 发送频控
// ─────────────────────────────────────────────

const (
	// SMSPhoneMinuteWindow 手机号分钟窗口（60 秒）
	SMSPhoneMinuteWindow = time.Minute
	// SMSPhoneMinuteLimit 同一手机号每分钟最多发送次数
	SMSPhoneMinuteLimit = 1
	// SMSPhoneDailyWindow 手机号日窗口（24 小时）
	SMSPhoneDailyWindow = 24 * time.Hour
	// SMSPhoneDailyLimit 同一手机号每天最多发送次数
	SMSPhoneDailyLimit = 5
	// SMSIPMinuteWindow IP 分钟窗口（60 秒）
	SMSIPMinuteWindow = time.Minute
	// SMSIPMinuteLimit 同一 IP 每分钟最多触发短信发送次数
	SMSIPMinuteLimit = 10
)

// KeySMSRatePhoneMinute 手机号分钟级频控计数器。
// 格式: tk:user:sms:rate:phone:minute:{phone}:{YYYYMMDDHHMI}
// TTL:  SMSPhoneMinuteWindow
// minute 格式例：202401011200
func KeySMSRatePhoneMinute(phone, minute string) string {
	return fmt.Sprintf("tk:user:sms:rate:phone:minute:%s:%s", phone, minute)
}

// KeySMSRatePhoneDaily 手机号日级频控计数器。
// 格式: tk:user:sms:rate:phone:daily:{phone}:{YYYYMMDD}
// TTL:  SMSPhoneDailyWindow
func KeySMSRatePhoneDaily(phone, day string) string {
	return fmt.Sprintf("tk:user:sms:rate:phone:daily:%s:%s", phone, day)
}

// KeySMSRateIPMinute IP 分钟级频控计数器（防代理批量攻击）。
// 格式: tk:user:sms:rate:ip:minute:{ip}:{YYYYMMDDHHMI}
// TTL:  SMSIPMinuteWindow
func KeySMSRateIPMinute(ip, minute string) string {
	return fmt.Sprintf("tk:user:sms:rate:ip:minute:%s:%s", ip, minute)
}

// ─────────────────────────────────────────────
// 登录失败频控（客户端用户）
// ─────────────────────────────────────────────

const (
	// LoginFailWindow 登录失败计数窗口（15 分钟）
	LoginFailWindow = 15 * time.Minute
	// LoginFailLimit 窗口内最大失败次数（超出后锁定账号）
	LoginFailLimit = 5
)

// KeyLoginFailPhone 客户端用户密码登录失败计数。
// 格式: tk:user:login:fail:phone:{phone}
// TTL:  LoginFailWindow（失败 5 次后锁定 15 分钟）
func KeyLoginFailPhone(phone string) string {
	return fmt.Sprintf("tk:user:login:fail:phone:%s", phone)
}

// KeyLoginFailIP 同 IP 登录失败计数（防撞库）。
// 格式: tk:user:login:fail:ip:{ip}
// TTL:  LoginFailWindow
func KeyLoginFailIP(ip string) string {
	return fmt.Sprintf("tk:user:login:fail:ip:%s", ip)
}

// ─────────────────────────────────────────────
// 分布式锁
// ─────────────────────────────────────────────

const (
	// LockRegisterTTL 注册锁最大持有时间
	LockRegisterTTL = 5 * time.Second
)

// KeyLockRegister 手机号注册分布式锁（防并发重复注册）。
// 格式: lock:user:register:{phone}
// TTL:  LockRegisterTTL
func KeyLockRegister(phone string) string {
	return fmt.Sprintf("lock:user:register:%s", phone)
}

// ─────────────────────────────────────────────
// 用户 Profile 缓存
// ─────────────────────────────────────────────

const (
	// UserProfileTTL 用户资料缓存有效期（15 分钟）
	UserProfileTTL = 15 * time.Minute
)

// KeyUserProfile 用户资料缓存。
// 格式: tk:user:profile:{userID}
// TTL:  UserProfileTTL（用户资料修改后主动 DEL）
func KeyUserProfile(userID uint64) string {
	return fmt.Sprintf("tk:user:profile:%d", userID)
}

// ─────────────────────────────────────────────
// 幂等去重
// ─────────────────────────────────────────────

const (
	// IdempotentTTL 幂等 key 有效期（60 秒）
	IdempotentTTL = 60 * time.Second
)

// KeyIdempotent 接口幂等去重（基于 X-Request-ID）。
// 格式: idempotent:user:{requestID}
// TTL:  IdempotentTTL
func KeyIdempotent(requestID string) string {
	return fmt.Sprintf("idempotent:user:%s", requestID)
}
