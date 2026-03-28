package repo

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/wangyahua6688-maker/tk-common/models"
	"github.com/wangyahua6688-maker/tk-common/utils/redisx/cmdx"
	redisx "github.com/wangyahua6688-maker/tk-common/utils/redisx/v8"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"tk-user/internal/constants/rediskey"
	"tk-user/internal/dto"

	"google.golang.org/grpc/metadata"
)

// 声明当前变量。
var (
	// errInvalidPhone 手机号格式错误。
	errInvalidPhone = errors.New("invalid phone number")
	// errInvalidPassword 密码不满足规则。
	errInvalidPassword = errors.New("invalid password")
	// errSMSCodeInvalid 验证码错误或已过期。
	errSMSCodeInvalid = errors.New("sms code invalid or expired")
	// errSMSServiceUnavailable 短信服务不可用（Redis 依赖不可用）。
	errSMSServiceUnavailable = errors.New("sms service unavailable")
	// errUserDisabled 用户状态不可用。
	errUserDisabled = errors.New("user disabled")
	// errLoginTooManyFailures 登录失败次数过多。
	errLoginTooManyFailures = errors.New("too many login failures, please try again later")
)

// SMSResult 为认证结果 DTO 的兼容别名。
type SMSResult = dto.SMSResult

// AuthResult 为认证结果 DTO 的兼容别名。
type AuthResult = dto.AuthResult

// phoneRegexp 中国大陆手机号校验规则（11 位且以 1 开头）。
var phoneRegexp = regexp.MustCompile(`^1\d{10}$`)

// SendSMSCode 发送短信验证码，内置手机号 + IP 双维度频控。
func (r *AuthRepository) SendSMSCode(ctx context.Context, phone string, purpose string) (SMSResult, error) {
	// 1) 归一化参数
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return SMSResult{}, errInvalidPhone
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if purpose == "" {
		purpose = "login"
	}

	// 2) 读取短信通道配置
	channel, err := r.loadActiveSMSChannel(ctx)
	if err != nil {
		return SMSResult{}, err
	}
	minuteLimit := channel.MinuteLimit
	if minuteLimit <= 0 {
		minuteLimit = 1
	}
	dailyLimit := channel.DailyLimit
	if dailyLimit <= 0 {
		dailyLimit = 20
	}
	codeTTL := r.smsCodeTTL
	if channel.CodeTTLSeconds > 0 {
		codeTTL = time.Duration(channel.CodeTTLSeconds) * time.Second
	}

	// 3) 执行手机号维度频控（分钟 + 日级）+ IP 维度频控
	clientIP := r.extractClientIP(ctx) // 从上下文提取客户端 IP
	if err := r.checkSMSRateLimit(ctx, phone, clientIP, minuteLimit, dailyLimit); err != nil {
		return SMSResult{}, err
	}

	// 4) Redis 不可用时直接失败，避免验证码链路 fail-open
	if r.redis == nil {
		return SMSResult{}, errSMSServiceUnavailable
	}

	// 5) 生成验证码并写入缓存
	code, err := generateSMSCode()
	if err != nil {
		return SMSResult{}, err
	}
	// 使用统一 key 定义
	smsKey := rediskey.KeySMSCode(purpose, phone)
	if err := redisx.SetString(ctx, r.redis, smsKey, code, codeTTL); err != nil {
		return SMSResult{}, err
	}

	// 6) 返回结果（mock 模式下携带预览码）
	result := SMSResult{
		Phone:      phone,
		Purpose:    purpose,
		ExpiresSec: int(codeTTL.Seconds()),
		MockMode:   channel.MockMode == 1,
	}
	if channel.MockMode == 1 && exposeSMSPreviewCode() {
		result.PreviewCode = code
	}
	return result, nil
}

// RegisterByPhone 手机号注册（含分布式锁防并发重复注册）。
func (r *AuthRepository) RegisterByPhone(ctx context.Context, phone string, password string, smsCode string, nickname string) (AuthResult, error) {
	// 1) 参数校验
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return AuthResult{}, errInvalidPhone
	}
	password = strings.TrimSpace(password)
	if len(password) < 6 || len(password) > 32 {
		return AuthResult{}, errInvalidPassword
	}

	// 2) 校验短信验证码
	smsCode = strings.TrimSpace(smsCode)
	if smsCode == "" {
		return AuthResult{}, errSMSCodeInvalid
	}
	ok, err := r.VerifySMSCode(ctx, phone, "register", smsCode)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errSMSCodeInvalid
	}

	// 3) 分布式锁：防止并发注册同一手机号（检查 + 插入之间的竞态窗口）
	lockKey := rediskey.KeyLockRegister(phone)
	var result AuthResult
	err = cmdx.TryLockFunc(ctx, r.redis, lockKey, rediskey.LockRegisterTTL, func() error {
		// 3.1) 在锁内再次检查手机号是否已存在（双重检查）
		existed := models.WUser{}
		dbErr := r.db.WithContext(ctx).Where("phone = ?", phone).First(&existed).Error
		if dbErr == nil && existed.ID > 0 {
			return fmt.Errorf("phone already registered")
		}
		if dbErr != nil && !errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return dbErr
		}

		// 3.2) 生成密码哈希并入库
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}
		displayName := strings.TrimSpace(nickname)
		if displayName == "" {
			displayName = "用户" + right(phone, 4)
		}
		user := models.WUser{
			Username:       "u" + phone,
			Phone:          phone,
			Nickname:       displayName,
			Avatar:         "",
			PasswordHash:   string(hash),
			RegisterSource: "password",
			UserType:       "natural",
			Status:         1,
		}
		if createErr := r.db.WithContext(ctx).Create(&user).Error; createErr != nil {
			return createErr
		}

		// 3.3) 签发 Token
		var issueErr error
		result, issueErr = r.issueSessionToken(ctx, user)
		return issueErr
	})

	if err != nil {
		// 区分锁获取失败和业务错误
		if errors.Is(err, cmdx.ErrLockNotAcquired) {
			return AuthResult{}, fmt.Errorf("registration in progress, please retry")
		}
		return AuthResult{}, err
	}
	return result, nil
}

// LoginByPassword 密码登录（含登录失败次数限制）。
func (r *AuthRepository) LoginByPassword(ctx context.Context, phone string, password string) (AuthResult, error) {
	// 1) 检查登录失败次数（手机号维度）
	if err := r.checkLoginFailLimit(ctx, phone); err != nil {
		return AuthResult{}, err
	}

	// 2) 读取用户
	user, err := r.findUserByPhone(ctx, phone)
	if err != nil {
		// 查询失败也记录失败次数，防止枚举用户是否存在
		_ = r.recordLoginFailure(ctx, phone)
		return AuthResult{}, err
	}
	if user.Status != 1 {
		return AuthResult{}, errUserDisabled
	}

	// 3) 校验密码
	if strings.TrimSpace(user.PasswordHash) == "" {
		return AuthResult{}, fmt.Errorf("password login unavailable")
	}
	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); bcryptErr != nil {
		// 密码错误：记录失败次数
		_ = r.recordLoginFailure(ctx, phone)
		return AuthResult{}, fmt.Errorf("phone or password incorrect")
	}

	// 4) 登录成功：清空失败计数
	_ = r.clearLoginFailure(ctx, phone)

	// 5) 签发 Token
	return r.issueSessionToken(ctx, *user)
}

// LoginBySMS 短信验证码登录（手机号不存在时自动注册）。
func (r *AuthRepository) LoginBySMS(ctx context.Context, phone string, smsCode string) (AuthResult, error) {
	// 1) 校验验证码
	ok, err := r.VerifySMSCode(ctx, phone, "login", smsCode)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errSMSCodeInvalid
	}

	// 2) 查询用户，不存在则自动创建
	user, err := r.findUserByPhone(ctx, phone)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AuthResult{}, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || user == nil {
		user = &models.WUser{
			Username:       "u" + phone,
			Phone:          phone,
			Nickname:       "用户" + right(phone, 4),
			UserType:       "natural",
			Status:         1,
			RegisterSource: "sms",
		}
		if createErr := r.db.WithContext(ctx).Create(user).Error; createErr != nil {
			return AuthResult{}, createErr
		}
	}
	if user.Status != 1 {
		return AuthResult{}, errUserDisabled
	}

	// 3) 签发 Token
	return r.issueSessionToken(ctx, *user)
}

// ProfileByToken 根据 AccessToken 读取用户资料（走 Redis 会话缓存）。
func (r *AuthRepository) ProfileByToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// 1) 归一化 token
	token := strings.TrimSpace(accessToken)
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	token = strings.TrimSpace(strings.TrimPrefix(token, "bearer "))
	if token == "" {
		return nil, fmt.Errorf("access token required")
	}

	// 2) 从 Redis 读取会话映射
	if r.redis == nil {
		return nil, fmt.Errorf("session storage unavailable")
	}
	key := rediskey.KeyAccessToken(token)
	userIDRaw, hit, err := redisx.GetString(ctx, r.redis, key)
	if err != nil {
		return nil, fmt.Errorf("access token invalid")
	}
	if !hit {
		return nil, fmt.Errorf("access token invalid")
	}
	userID64, err := strconv.ParseUint(strings.TrimSpace(userIDRaw), 10, 64)
	if err != nil || userID64 == 0 {
		return nil, fmt.Errorf("access token invalid")
	}

	// 3) 查询用户资料并返回
	user := models.WUser{}
	if err := r.db.WithContext(ctx).Where("id = ?", userID64).First(&user).Error; err != nil {
		return nil, err
	}
	return buildUserProfile(user), nil
}

// VerifySMSCode 校验验证码（校验通过后立即删除，防重放）。
func (r *AuthRepository) VerifySMSCode(ctx context.Context, phone string, purpose string, code string) (bool, error) {
	// 1) 参数校验
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return false, errInvalidPhone
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if purpose == "" {
		purpose = "login"
	}
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false, nil
	}

	// 2) Redis 读取缓存验证码
	if r.redis == nil {
		return false, errSMSServiceUnavailable
	}
	smsKey := rediskey.KeySMSCode(purpose, phone)
	cached, hit, err := redisx.GetString(ctx, r.redis, smsKey)
	if err != nil {
		return false, err
	}
	if !hit {
		return false, nil
	}
	if strings.TrimSpace(cached) != code {
		return false, nil
	}

	// 3) 校验通过后立即删除验证码，防止重复使用（一次性语义）
	_ = redisx.Del(ctx, r.redis, smsKey)
	return true, nil
}

// ─────────────────────────────────────────────
// 私有方法
// ─────────────────────────────────────────────

// checkSMSRateLimit 执行短信发送频控（手机号分钟 + 日级 + IP 分钟）。
func (r *AuthRepository) checkSMSRateLimit(ctx context.Context, phone, clientIP string, minuteLimit int, dailyLimit int) error {
	if r.redis == nil {
		return errSMSServiceUnavailable
	}

	now := time.Now()
	minuteStr := now.Format("200601021504")
	dayStr := now.Format("20060102")

	// 构建联合限流规则
	rules := []cmdx.RateLimitRule{
		{
			// 手机号分钟频控：同号 1 分钟内最多发 N 次
			Key:    rediskey.KeySMSRatePhoneMinute(phone, minuteStr),
			Limit:  int64(minuteLimit),
			Window: rediskey.SMSPhoneMinuteWindow,
		},
		{
			// 手机号日频控：同号每天最多发 N 次
			Key:    rediskey.KeySMSRatePhoneDaily(phone, dayStr),
			Limit:  int64(dailyLimit),
			Window: rediskey.SMSPhoneDailyWindow,
		},
	}

	// IP 维度频控（客户端 IP 存在时追加）
	if clientIP != "" && clientIP != "unknown" {
		rules = append(rules, cmdx.RateLimitRule{
			Key:    rediskey.KeySMSRateIPMinute(clientIP, minuteStr),
			Limit:  rediskey.SMSIPMinuteLimit,
			Window: rediskey.SMSIPMinuteWindow,
		})
	}

	result, err := cmdx.MultiWindowAllow(ctx, r.redis, rules)
	if err != nil {
		// Redis 通信错误时 fail-open（记录日志，不阻断发送）
		return nil
	}
	if !result.Allowed {
		if result.BlockedRule != nil && strings.Contains(result.BlockedRule.Key, "daily") {
			return fmt.Errorf("sms daily limit reached")
		}
		return fmt.Errorf("sms send too frequent")
	}
	return nil
}

// checkLoginFailLimit 检查密码登录失败次数是否超限。
func (r *AuthRepository) checkLoginFailLimit(ctx context.Context, phone string) error {
	if r.redis == nil {
		return nil // Redis 不可用时跳过限制（fail-open）
	}

	key := rediskey.KeyLoginFailPhone(phone)
	val, hit, err := redisx.GetString(ctx, r.redis, key)
	if err != nil || !hit {
		return nil
	}
	count, _ := strconv.Atoi(val)
	if count >= rediskey.LoginFailLimit {
		return errLoginTooManyFailures
	}
	return nil
}

// recordLoginFailure 记录一次密码登录失败。
func (r *AuthRepository) recordLoginFailure(ctx context.Context, phone string) error {
	if r.redis == nil {
		return nil
	}
	key := rediskey.KeyLoginFailPhone(phone)
	_, err := cmdx.Incr(ctx, r.redis, key, rediskey.LoginFailWindow)
	return err
}

// clearLoginFailure 登录成功后清空失败计数。
func (r *AuthRepository) clearLoginFailure(ctx context.Context, phone string) error {
	if r.redis == nil {
		return nil
	}
	key := rediskey.KeyLoginFailPhone(phone)
	_, err := cmdx.Del(ctx, r.redis, key)
	return err
}

// extractClientIP 从 context 中提取客户端 IP（由 gRPC 元数据或 HTTP 中间件注入）。
// 若未注入则返回空字符串（限流将跳过 IP 维度）。
func (r *AuthRepository) extractClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	// 约定：调用方通过 context.WithValue(ctx, "client_ip", ip) 注入
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return strings.TrimSpace(ip)
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for _, key := range []string{"x-client-ip", "client_ip"} {
			values := md.Get(key)
			if len(values) > 0 {
				if ip := strings.TrimSpace(values[0]); ip != "" {
					return ip
				}
			}
		}
	}
	return ""
}

// issueSessionToken 生成访问令牌与刷新令牌并更新用户登录时间。
func (r *AuthRepository) issueSessionToken(ctx context.Context, user models.WUser) (AuthResult, error) {
	// 1) 生成随机 Token（UUID 去掉横线）
	accessToken := strings.ReplaceAll(uuid.NewString(), "-", "")
	refreshToken := strings.ReplaceAll(uuid.NewString(), "-", "")

	// 2) 写入 Redis 会话存储（使用统一 key 定义）
	if r.redis == nil {
		return AuthResult{}, errSMSServiceUnavailable
	}
	userIDStr := strconv.FormatUint(uint64(user.ID), 10)

	accessKey := rediskey.KeyAccessToken(accessToken)
	refreshKey := rediskey.KeyRefreshToken(refreshToken)
	pairKey := rediskey.KeyAccessRefreshPair(accessToken)

	if err := redisx.SetString(ctx, r.redis, accessKey, userIDStr, r.accessTokenTTL); err != nil {
		return AuthResult{}, err
	}
	if err := redisx.SetString(ctx, r.redis, refreshKey, userIDStr, r.refreshTokenTTL); err != nil {
		return AuthResult{}, err
	}
	// 记录 access → refresh 映射，便于主动登出时联动失效
	_ = redisx.SetString(ctx, r.redis, pairKey, refreshToken, r.accessTokenTTL)

	// 3) 更新数据库中的最后登录时间
	now := time.Now()
	_ = r.db.WithContext(ctx).
		Model(&models.WUser{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
	user.LastLoginAt = &now

	// 4) 返回统一响应结构
	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         buildUserProfile(user),
	}, nil
}

// loadActiveSMSChannel 读取当前启用的短信通道配置。
func (r *AuthRepository) loadActiveSMSChannel(ctx context.Context) (models.WSMSChannel, error) {
	channel := models.WSMSChannel{}
	err := r.db.WithContext(ctx).
		Where("status = 1").
		Order("id ASC").
		First(&channel).Error
	if err == nil {
		return channel, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 无配置时返回默认 mock 通道
		return models.WSMSChannel{
			Provider:       "mock",
			ChannelName:    "默认模拟通道",
			MinuteLimit:    1,
			DailyLimit:     20,
			CodeTTLSeconds: int(r.smsCodeTTL.Seconds()),
			MockMode:       1,
			Status:         1,
		}, nil
	}
	return models.WSMSChannel{}, err
}

// findUserByPhone 按手机号查询用户。
func (r *AuthRepository) findUserByPhone(ctx context.Context, phone string) (*models.WUser, error) {
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return nil, errInvalidPhone
	}
	row := models.WUser{}
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// buildUserProfile 统一组装用户资料（不含敏感字段如密码哈希）。
func buildUserProfile(user models.WUser) map[string]interface{} {
	return map[string]interface{}{
		"id":              user.ID,
		"username":        user.Username,
		"phone":           user.Phone,
		"nickname":        user.Nickname,
		"avatar":          user.Avatar,
		"user_type":       user.UserType,
		"status":          user.Status,
		"register_source": user.RegisterSource,
		"last_login_at":   user.LastLoginAt,
	}
}

// generateSMSCode 使用密码学安全随机源生成 6 位验证码。
func generateSMSCode() (string, error) {
	upper := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, upper)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// exposeSMSPreviewCode 控制是否允许在响应中返回测试验证码（仅开发环境）。
func exposeSMSPreviewCode() bool {
	flag := strings.ToLower(strings.TrimSpace(os.Getenv("TK_USER_EXPOSE_SMS_PREVIEW")))
	return flag == "1" || flag == "true" || flag == "yes" || flag == "on"
}

// right 返回字符串右侧 n 位字符。
func right(raw string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(raw) <= n {
		return raw
	}
	return raw[len(raw)-n:]
}
