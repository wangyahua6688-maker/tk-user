package repo

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"tk-common/models"
	redisx "tk-common/utils/redisx/v8"
)

var (
	// errInvalidPhone 手机号格式错误。
	errInvalidPhone = errors.New("invalid phone number")
	// errInvalidPassword 密码不满足规则。
	errInvalidPassword = errors.New("invalid password")
	// errSMSCodeInvalid 验证码错误或已过期。
	errSMSCodeInvalid = errors.New("sms code invalid or expired")
	// errUserDisabled 用户状态不可用。
	errUserDisabled = errors.New("user disabled")
)

// SMSResult 短信发送响应。
type SMSResult struct {
	Phone       string `json:"phone"`
	Purpose     string `json:"purpose"`
	ExpiresSec  int    `json:"expires_sec"`
	MockMode    bool   `json:"mock_mode"`
	PreviewCode string `json:"preview_code,omitempty"`
}

// AuthResult 用户登录/注册成功后的统一返回结构。
type AuthResult struct {
	AccessToken  string                 `json:"access_token"`
	RefreshToken string                 `json:"refresh_token"`
	User         map[string]interface{} `json:"user"`
}

// phoneRegexp 中国大陆手机号校验规则（11 位且以 1 开头）。
var phoneRegexp = regexp.MustCompile(`^1\d{10}$`)

// SendSMSCode 发送短信验证码（当前支持 mock 模式）。
func (r *Repository) SendSMSCode(ctx context.Context, phone string, purpose string) (SMSResult, error) {
	// 1) 归一化手机号与用途参数。
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return SMSResult{}, errInvalidPhone
	}
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if purpose == "" {
		purpose = "login"
	}

	// 2) 读取短信通道配置；若未配置则使用默认 mock 配置。
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

	// 3) 执行分钟级与日级频控。
	if err := r.checkSMSRateLimit(ctx, phone, minuteLimit, dailyLimit); err != nil {
		return SMSResult{}, err
	}

	// 4) 生成验证码并写入缓存。
	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	if r.redis != nil {
		smsKey := fmt.Sprintf("tk:user:sms:%s:%s", purpose, phone)
		if err := redisx.SetString(ctx, r.redis, smsKey, code, codeTTL); err != nil {
			return SMSResult{}, err
		}
	}

	// 5) 当前阶段先返回 mock 预览码；正式短信网关可在此替换发送逻辑。
	result := SMSResult{
		Phone:      phone,
		Purpose:    purpose,
		ExpiresSec: int(codeTTL.Seconds()),
		MockMode:   channel.MockMode == 1,
	}
	if channel.MockMode == 1 {
		result.PreviewCode = code
	}
	return result, nil
}

// RegisterByPhone 手机号注册。
func (r *Repository) RegisterByPhone(ctx context.Context, phone string, password string, smsCode string, nickname string) (AuthResult, error) {
	// 1) 参数校验。
	phone = strings.TrimSpace(phone)
	if !phoneRegexp.MatchString(phone) {
		return AuthResult{}, errInvalidPhone
	}
	password = strings.TrimSpace(password)
	if len(password) < 6 || len(password) > 32 {
		return AuthResult{}, errInvalidPassword
	}

	// 2) 校验验证码（允许后台管理场景传空码以跳过）。
	if strings.TrimSpace(smsCode) != "" {
		ok, err := r.VerifySMSCode(ctx, phone, "register", smsCode)
		if err != nil {
			return AuthResult{}, err
		}
		if !ok {
			return AuthResult{}, errSMSCodeInvalid
		}
	}

	// 3) 校验手机号是否已存在。
	existed := models.WUser{}
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&existed).Error
	if err == nil && existed.ID > 0 {
		return AuthResult{}, fmt.Errorf("phone already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return AuthResult{}, err
	}

	// 4) 生成密码哈希并入库。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, err
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
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return AuthResult{}, err
	}

	// 5) 创建登录态令牌并返回资料。
	return r.issueSessionToken(ctx, user)
}

// LoginByPassword 密码登录。
func (r *Repository) LoginByPassword(ctx context.Context, phone string, password string) (AuthResult, error) {
	// 1) 读取用户。
	user, err := r.findUserByPhone(ctx, phone)
	if err != nil {
		return AuthResult{}, err
	}
	if user.Status != 1 {
		return AuthResult{}, errUserDisabled
	}

	// 2) 校验密码。
	if strings.TrimSpace(user.PasswordHash) == "" {
		return AuthResult{}, fmt.Errorf("password login unavailable")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return AuthResult{}, fmt.Errorf("phone or password incorrect")
	}

	// 3) 签发登录令牌。
	return r.issueSessionToken(ctx, *user)
}

// LoginBySMS 短信验证码登录（手机号不存在时自动注册）。
func (r *Repository) LoginBySMS(ctx context.Context, phone string, smsCode string) (AuthResult, error) {
	// 1) 校验验证码。
	ok, err := r.VerifySMSCode(ctx, phone, "login", smsCode)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errSMSCodeInvalid
	}

	// 2) 查询用户，不存在则自动创建自然用户。
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

	// 3) 签发登录令牌。
	return r.issueSessionToken(ctx, *user)
}

// ProfileByToken 根据 access token 读取用户资料。
func (r *Repository) ProfileByToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// 1) 归一化 token。
	token := strings.TrimSpace(accessToken)
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	token = strings.TrimSpace(strings.TrimPrefix(token, "bearer "))
	if token == "" {
		return nil, fmt.Errorf("access token required")
	}

	// 2) 从 Redis 读取会话映射。
	if r.redis == nil {
		return nil, fmt.Errorf("session storage unavailable")
	}
	key := fmt.Sprintf("tk:user:access:%s", token)
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

	// 3) 查询用户资料并返回。
	user := models.WUser{}
	if err := r.db.WithContext(ctx).Where("id = ?", userID64).First(&user).Error; err != nil {
		return nil, err
	}
	return buildUserProfile(user), nil
}

// VerifySMSCode 校验验证码。
func (r *Repository) VerifySMSCode(ctx context.Context, phone string, purpose string, code string) (bool, error) {
	// 1) 参数校验。
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

	// 2) Redis 读取缓存验证码。
	if r.redis == nil {
		// 无 Redis 时不启用验证码校验，避免阻塞联调。
		return true, nil
	}
	smsKey := fmt.Sprintf("tk:user:sms:%s:%s", purpose, phone)
	cached, hit, err := redisx.GetString(ctx, r.redis, smsKey)
	if err != nil {
		return false, nil
	}
	if !hit {
		return false, nil
	}
	if strings.TrimSpace(cached) != code {
		return false, nil
	}

	// 3) 校验通过后删除验证码，防止重复使用。
	_ = redisx.Del(ctx, r.redis, smsKey)
	return true, nil
}

// checkSMSRateLimit 执行短信发送频控。
func (r *Repository) checkSMSRateLimit(ctx context.Context, phone string, minuteLimit int, dailyLimit int) error {
	if r.redis == nil {
		return nil
	}

	now := time.Now()
	minuteKey := fmt.Sprintf("tk:user:sms:minute:%s:%s", phone, now.Format("200601021504"))
	dailyKey := fmt.Sprintf("tk:user:sms:daily:%s:%s", phone, now.Format("20060102"))

	minuteCount, err := redisx.IncrWithExpire(ctx, r.redis, minuteKey, time.Minute)
	if err != nil {
		return err
	}
	if int(minuteCount) > minuteLimit {
		return fmt.Errorf("sms send too frequent")
	}

	dailyCount, err := redisx.IncrWithExpire(ctx, r.redis, dailyKey, 24*time.Hour)
	if err != nil {
		return err
	}
	if int(dailyCount) > dailyLimit {
		return fmt.Errorf("sms daily limit reached")
	}
	return nil
}

// loadActiveSMSChannel 读取当前启用的短信通道。
func (r *Repository) loadActiveSMSChannel(ctx context.Context) (models.WSMSChannel, error) {
	channel := models.WSMSChannel{}
	err := r.db.WithContext(ctx).
		Where("status = 1").
		Order("id ASC").
		First(&channel).Error
	if err == nil {
		return channel, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有配置时默认启用 mock 通道，保障开发联调。
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

// issueSessionToken 生成访问令牌与刷新令牌并刷新用户登录时间。
func (r *Repository) issueSessionToken(ctx context.Context, user models.WUser) (AuthResult, error) {
	// 1) 生成 token。
	accessToken := strings.ReplaceAll(uuid.NewString(), "-", "")
	refreshToken := strings.ReplaceAll(uuid.NewString(), "-", "")

	// 2) 写入 Redis 会话存储。
	if r.redis != nil {
		accessKey := fmt.Sprintf("tk:user:access:%s", accessToken)
		refreshKey := fmt.Sprintf("tk:user:refresh:%s", refreshToken)
		if err := redisx.SetString(ctx, r.redis, accessKey, strconv.FormatUint(uint64(user.ID), 10), r.accessTokenTTL); err != nil {
			return AuthResult{}, err
		}
		if err := redisx.SetString(ctx, r.redis, refreshKey, strconv.FormatUint(uint64(user.ID), 10), r.refreshTokenTTL); err != nil {
			return AuthResult{}, err
		}
		// 记录 access->refresh 关系，便于后续扩展注销逻辑。
		pairKey := fmt.Sprintf("tk:user:access-refresh:%s", accessToken)
		_ = redisx.SetString(ctx, r.redis, pairKey, refreshToken, r.accessTokenTTL)
	}

	// 3) 更新登录时间。
	now := time.Now()
	_ = r.db.WithContext(ctx).
		Model(&models.WUser{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
	user.LastLoginAt = &now

	// 4) 返回登录态结构。
	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         buildUserProfile(user),
	}, nil
}

// findUserByPhone 按手机号查询用户。
func (r *Repository) findUserByPhone(ctx context.Context, phone string) (*models.WUser, error) {
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

// buildUserProfile 统一组装用户资料。
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
