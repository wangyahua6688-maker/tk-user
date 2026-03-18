package repo

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
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
)

// SMSResult 短信发送响应。
type SMSResult struct {
	// 处理当前语句逻辑。
	Phone string `json:"phone"`
	// 处理当前语句逻辑。
	Purpose string `json:"purpose"`
	// 处理当前语句逻辑。
	ExpiresSec int `json:"expires_sec"`
	// 处理当前语句逻辑。
	MockMode bool `json:"mock_mode"`
	// 处理当前语句逻辑。
	PreviewCode string `json:"preview_code,omitempty"`
}

// AuthResult 用户登录/注册成功后的统一返回结构。
type AuthResult struct {
	// 处理当前语句逻辑。
	AccessToken string `json:"access_token"`
	// 处理当前语句逻辑。
	RefreshToken string `json:"refresh_token"`
	// 处理当前语句逻辑。
	User map[string]interface{} `json:"user"`
}

// phoneRegexp 中国大陆手机号校验规则（11 位且以 1 开头）。
var phoneRegexp = regexp.MustCompile(`^1\d{10}$`)

// SendSMSCode 发送短信验证码（当前支持 mock 模式）。
func (r *Repository) SendSMSCode(ctx context.Context, phone string, purpose string) (SMSResult, error) {
	// 1) 归一化手机号与用途参数。
	phone = strings.TrimSpace(phone)
	// 判断条件并进入对应分支逻辑。
	if !phoneRegexp.MatchString(phone) {
		// 返回当前处理结果。
		return SMSResult{}, errInvalidPhone
	}
	// 更新当前变量或字段值。
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	// 判断条件并进入对应分支逻辑。
	if purpose == "" {
		// 更新当前变量或字段值。
		purpose = "login"
	}

	// 2) 读取短信通道配置；若未配置则使用默认 mock 配置。
	channel, err := r.loadActiveSMSChannel(ctx)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return SMSResult{}, err
	}
	// 定义并初始化当前变量。
	minuteLimit := channel.MinuteLimit
	// 判断条件并进入对应分支逻辑。
	if minuteLimit <= 0 {
		// 更新当前变量或字段值。
		minuteLimit = 1
	}
	// 定义并初始化当前变量。
	dailyLimit := channel.DailyLimit
	// 判断条件并进入对应分支逻辑。
	if dailyLimit <= 0 {
		// 更新当前变量或字段值。
		dailyLimit = 20
	}
	// 定义并初始化当前变量。
	codeTTL := r.smsCodeTTL
	// 判断条件并进入对应分支逻辑。
	if channel.CodeTTLSeconds > 0 {
		// 更新当前变量或字段值。
		codeTTL = time.Duration(channel.CodeTTLSeconds) * time.Second
	}

	// 3) 执行分钟级与日级频控。
	if err := r.checkSMSRateLimit(ctx, phone, minuteLimit, dailyLimit); err != nil {
		// 返回当前处理结果。
		return SMSResult{}, err
	}

	// 4) Redis 不可用时直接失败，避免验证码链路 fail-open。
	if r.redis == nil {
		// 返回当前处理结果。
		return SMSResult{}, errSMSServiceUnavailable
	}
	// 5) 生成验证码并写入缓存。
	code, err := generateSMSCode()
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return SMSResult{}, err
	}
	// 定义并初始化当前变量。
	smsKey := fmt.Sprintf("tk:user:sms:%s:%s", purpose, phone)
	// 判断条件并进入对应分支逻辑。
	if err := redisx.SetString(ctx, r.redis, smsKey, code, codeTTL); err != nil {
		// 返回当前处理结果。
		return SMSResult{}, err
	}

	// 6) 当前阶段先返回 mock 预览码；正式短信网关可在此替换发送逻辑。
	result := SMSResult{
		// 处理当前语句逻辑。
		Phone: phone,
		// 处理当前语句逻辑。
		Purpose: purpose,
		// 调用int完成当前处理。
		ExpiresSec: int(codeTTL.Seconds()),
		// 处理当前语句逻辑。
		MockMode: channel.MockMode == 1,
	}
	// 判断条件并进入对应分支逻辑。
	if channel.MockMode == 1 && exposeSMSPreviewCode() {
		// 更新当前变量或字段值。
		result.PreviewCode = code
	}
	// 返回当前处理结果。
	return result, nil
}

// RegisterByPhone 手机号注册。
func (r *Repository) RegisterByPhone(ctx context.Context, phone string, password string, smsCode string, nickname string) (AuthResult, error) {
	// 1) 参数校验。
	phone = strings.TrimSpace(phone)
	// 判断条件并进入对应分支逻辑。
	if !phoneRegexp.MatchString(phone) {
		// 返回当前处理结果。
		return AuthResult{}, errInvalidPhone
	}
	// 更新当前变量或字段值。
	password = strings.TrimSpace(password)
	// 判断条件并进入对应分支逻辑。
	if len(password) < 6 || len(password) > 32 {
		// 返回当前处理结果。
		return AuthResult{}, errInvalidPassword
	}

	// 2) 注册必须校验短信验证码，空码直接拒绝。
	smsCode = strings.TrimSpace(smsCode)
	// 判断条件并进入对应分支逻辑。
	if smsCode == "" {
		// 返回当前处理结果。
		return AuthResult{}, errSMSCodeInvalid
	}
	// 定义并初始化当前变量。
	ok, err := r.VerifySMSCode(ctx, phone, "register", smsCode)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 判断条件并进入对应分支逻辑。
	if !ok {
		// 返回当前处理结果。
		return AuthResult{}, errSMSCodeInvalid
	}

	// 3) 校验手机号是否已存在。
	existed := models.WUser{}
	// 定义并初始化当前变量。
	err = r.db.WithContext(ctx).Where("phone = ?", phone).First(&existed).Error
	// 判断条件并进入对应分支逻辑。
	if err == nil && existed.ID > 0 {
		// 返回当前处理结果。
		return AuthResult{}, fmt.Errorf("phone already registered")
	}
	// 判断条件并进入对应分支逻辑。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 返回当前处理结果。
		return AuthResult{}, err
	}

	// 4) 生成密码哈希并入库。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 定义并初始化当前变量。
	displayName := strings.TrimSpace(nickname)
	// 判断条件并进入对应分支逻辑。
	if displayName == "" {
		// 更新当前变量或字段值。
		displayName = "用户" + right(phone, 4)
	}
	// 定义并初始化当前变量。
	user := models.WUser{
		// 处理当前语句逻辑。
		Username: "u" + phone,
		// 处理当前语句逻辑。
		Phone: phone,
		// 处理当前语句逻辑。
		Nickname: displayName,
		// 处理当前语句逻辑。
		Avatar: "",
		// 调用string完成当前处理。
		PasswordHash: string(hash),
		// 处理当前语句逻辑。
		RegisterSource: "password",
		// 处理当前语句逻辑。
		UserType: "natural",
		// 处理当前语句逻辑。
		Status: 1,
	}
	// 判断条件并进入对应分支逻辑。
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}

	// 5) 创建登录态令牌并返回资料。
	return r.issueSessionToken(ctx, user)
}

// LoginByPassword 密码登录。
func (r *Repository) LoginByPassword(ctx context.Context, phone string, password string) (AuthResult, error) {
	// 1) 读取用户。
	user, err := r.findUserByPhone(ctx, phone)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 判断条件并进入对应分支逻辑。
	if user.Status != 1 {
		// 返回当前处理结果。
		return AuthResult{}, errUserDisabled
	}

	// 2) 校验密码。
	if strings.TrimSpace(user.PasswordHash) == "" {
		// 返回当前处理结果。
		return AuthResult{}, fmt.Errorf("password login unavailable")
	}
	// 判断条件并进入对应分支逻辑。
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		// 返回当前处理结果。
		return AuthResult{}, fmt.Errorf("phone or password incorrect")
	}

	// 3) 签发登录令牌。
	return r.issueSessionToken(ctx, *user)
}

// LoginBySMS 短信验证码登录（手机号不存在时自动注册）。
func (r *Repository) LoginBySMS(ctx context.Context, phone string, smsCode string) (AuthResult, error) {
	// 1) 校验验证码。
	ok, err := r.VerifySMSCode(ctx, phone, "login", smsCode)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 判断条件并进入对应分支逻辑。
	if !ok {
		// 返回当前处理结果。
		return AuthResult{}, errSMSCodeInvalid
	}

	// 2) 查询用户，不存在则自动创建自然用户。
	user, err := r.findUserByPhone(ctx, phone)
	// 判断条件并进入对应分支逻辑。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 判断条件并进入对应分支逻辑。
	if errors.Is(err, gorm.ErrRecordNotFound) || user == nil {
		// 更新当前变量或字段值。
		user = &models.WUser{
			// 处理当前语句逻辑。
			Username: "u" + phone,
			// 处理当前语句逻辑。
			Phone: phone,
			// 调用right完成当前处理。
			Nickname: "用户" + right(phone, 4),
			// 处理当前语句逻辑。
			UserType: "natural",
			// 处理当前语句逻辑。
			Status: 1,
			// 处理当前语句逻辑。
			RegisterSource: "sms",
		}
		// 判断条件并进入对应分支逻辑。
		if createErr := r.db.WithContext(ctx).Create(user).Error; createErr != nil {
			// 返回当前处理结果。
			return AuthResult{}, createErr
		}
	}
	// 判断条件并进入对应分支逻辑。
	if user.Status != 1 {
		// 返回当前处理结果。
		return AuthResult{}, errUserDisabled
	}

	// 3) 签发登录令牌。
	return r.issueSessionToken(ctx, *user)
}

// ProfileByToken 根据 access token 读取用户资料。
func (r *Repository) ProfileByToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	// 1) 归一化 token。
	token := strings.TrimSpace(accessToken)
	// 更新当前变量或字段值。
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	// 更新当前变量或字段值。
	token = strings.TrimSpace(strings.TrimPrefix(token, "bearer "))
	// 判断条件并进入对应分支逻辑。
	if token == "" {
		// 返回当前处理结果。
		return nil, fmt.Errorf("access token required")
	}

	// 2) 从 Redis 读取会话映射。
	if r.redis == nil {
		// 返回当前处理结果。
		return nil, fmt.Errorf("session storage unavailable")
	}
	// 定义并初始化当前变量。
	key := fmt.Sprintf("tk:user:access:%s", token)
	// 定义并初始化当前变量。
	userIDRaw, hit, err := redisx.GetString(ctx, r.redis, key)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, fmt.Errorf("access token invalid")
	}
	// 判断条件并进入对应分支逻辑。
	if !hit {
		// 返回当前处理结果。
		return nil, fmt.Errorf("access token invalid")
	}
	// 定义并初始化当前变量。
	userID64, err := strconv.ParseUint(strings.TrimSpace(userIDRaw), 10, 64)
	// 判断条件并进入对应分支逻辑。
	if err != nil || userID64 == 0 {
		// 返回当前处理结果。
		return nil, fmt.Errorf("access token invalid")
	}

	// 3) 查询用户资料并返回。
	user := models.WUser{}
	// 判断条件并进入对应分支逻辑。
	if err := r.db.WithContext(ctx).Where("id = ?", userID64).First(&user).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}
	// 返回当前处理结果。
	return buildUserProfile(user), nil
}

// VerifySMSCode 校验验证码。
func (r *Repository) VerifySMSCode(ctx context.Context, phone string, purpose string, code string) (bool, error) {
	// 1) 参数校验。
	phone = strings.TrimSpace(phone)
	// 判断条件并进入对应分支逻辑。
	if !phoneRegexp.MatchString(phone) {
		// 返回当前处理结果。
		return false, errInvalidPhone
	}
	// 更新当前变量或字段值。
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	// 判断条件并进入对应分支逻辑。
	if purpose == "" {
		// 更新当前变量或字段值。
		purpose = "login"
	}
	// 更新当前变量或字段值。
	code = strings.TrimSpace(code)
	// 判断条件并进入对应分支逻辑。
	if len(code) != 6 {
		// 返回当前处理结果。
		return false, nil
	}

	// 2) Redis 读取缓存验证码。
	if r.redis == nil {
		// 返回当前处理结果。
		return false, errSMSServiceUnavailable
	}
	// 定义并初始化当前变量。
	smsKey := fmt.Sprintf("tk:user:sms:%s:%s", purpose, phone)
	// 定义并初始化当前变量。
	cached, hit, err := redisx.GetString(ctx, r.redis, smsKey)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return false, err
	}
	// 判断条件并进入对应分支逻辑。
	if !hit {
		// 返回当前处理结果。
		return false, nil
	}
	// 判断条件并进入对应分支逻辑。
	if strings.TrimSpace(cached) != code {
		// 返回当前处理结果。
		return false, nil
	}

	// 3) 校验通过后删除验证码，防止重复使用。
	_ = redisx.Del(ctx, r.redis, smsKey)
	// 返回当前处理结果。
	return true, nil
}

// checkSMSRateLimit 执行短信发送频控。
func (r *Repository) checkSMSRateLimit(ctx context.Context, phone string, minuteLimit int, dailyLimit int) error {
	// 判断条件并进入对应分支逻辑。
	if r.redis == nil {
		// 返回当前处理结果。
		return errSMSServiceUnavailable
	}

	// 定义并初始化当前变量。
	now := time.Now()
	// 定义并初始化当前变量。
	minuteKey := fmt.Sprintf("tk:user:sms:minute:%s:%s", phone, now.Format("200601021504"))
	// 定义并初始化当前变量。
	dailyKey := fmt.Sprintf("tk:user:sms:daily:%s:%s", phone, now.Format("20060102"))

	// 定义并初始化当前变量。
	minuteCount, err := redisx.IncrWithExpire(ctx, r.redis, minuteKey, time.Minute)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return err
	}
	// 判断条件并进入对应分支逻辑。
	if int(minuteCount) > minuteLimit {
		// 返回当前处理结果。
		return fmt.Errorf("sms send too frequent")
	}

	// 定义并初始化当前变量。
	dailyCount, err := redisx.IncrWithExpire(ctx, r.redis, dailyKey, 24*time.Hour)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return err
	}
	// 判断条件并进入对应分支逻辑。
	if int(dailyCount) > dailyLimit {
		// 返回当前处理结果。
		return fmt.Errorf("sms daily limit reached")
	}
	// 返回当前处理结果。
	return nil
}

// loadActiveSMSChannel 读取当前启用的短信通道。
func (r *Repository) loadActiveSMSChannel(ctx context.Context) (models.WSMSChannel, error) {
	// 定义并初始化当前变量。
	channel := models.WSMSChannel{}
	// 定义并初始化当前变量。
	err := r.db.WithContext(ctx).
		// 更新当前变量或字段值。
		Where("status = 1").
		// 调用Order完成当前处理。
		Order("id ASC").
		// 调用First完成当前处理。
		First(&channel).Error
	// 判断条件并进入对应分支逻辑。
	if err == nil {
		// 返回当前处理结果。
		return channel, nil
	}
	// 判断条件并进入对应分支逻辑。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 没有配置时默认启用 mock 通道，保障开发联调。
		return models.WSMSChannel{
			// 处理当前语句逻辑。
			Provider: "mock",
			// 处理当前语句逻辑。
			ChannelName: "默认模拟通道",
			// 处理当前语句逻辑。
			MinuteLimit: 1,
			// 处理当前语句逻辑。
			DailyLimit: 20,
			// 调用int完成当前处理。
			CodeTTLSeconds: int(r.smsCodeTTL.Seconds()),
			// 处理当前语句逻辑。
			MockMode: 1,
			// 处理当前语句逻辑。
			Status: 1,
			// 处理当前语句逻辑。
		}, nil
	}
	// 返回当前处理结果。
	return models.WSMSChannel{}, err
}

// issueSessionToken 生成访问令牌与刷新令牌并刷新用户登录时间。
func (r *Repository) issueSessionToken(ctx context.Context, user models.WUser) (AuthResult, error) {
	// 1) 生成 token。
	accessToken := strings.ReplaceAll(uuid.NewString(), "-", "")
	// 定义并初始化当前变量。
	refreshToken := strings.ReplaceAll(uuid.NewString(), "-", "")

	// 2) 写入 Redis 会话存储。
	if r.redis == nil {
		// 返回当前处理结果。
		return AuthResult{}, errSMSServiceUnavailable
	}
	// 定义并初始化当前变量。
	accessKey := fmt.Sprintf("tk:user:access:%s", accessToken)
	// 定义并初始化当前变量。
	refreshKey := fmt.Sprintf("tk:user:refresh:%s", refreshToken)
	// 判断条件并进入对应分支逻辑。
	if err := redisx.SetString(ctx, r.redis, accessKey, strconv.FormatUint(uint64(user.ID), 10), r.accessTokenTTL); err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 判断条件并进入对应分支逻辑。
	if err := redisx.SetString(ctx, r.redis, refreshKey, strconv.FormatUint(uint64(user.ID), 10), r.refreshTokenTTL); err != nil {
		// 返回当前处理结果。
		return AuthResult{}, err
	}
	// 记录 access->refresh 关系，便于后续扩展注销逻辑。
	pairKey := fmt.Sprintf("tk:user:access-refresh:%s", accessToken)
	// 更新当前变量或字段值。
	_ = redisx.SetString(ctx, r.redis, pairKey, refreshToken, r.accessTokenTTL)

	// 3) 更新登录时间。
	now := time.Now()
	// 更新当前变量或字段值。
	_ = r.db.WithContext(ctx).
		// 调用Model完成当前处理。
		Model(&models.WUser{}).
		// 更新当前变量或字段值。
		Where("id = ?", user.ID).
		// 调用Updates完成当前处理。
		Updates(map[string]interface{}{
			// 处理当前语句逻辑。
			"last_login_at": now,
			// 处理当前语句逻辑。
			"updated_at": now,
			// 处理当前语句逻辑。
		}).Error
	// 更新当前变量或字段值。
	user.LastLoginAt = &now

	// 4) 返回登录态结构。
	return AuthResult{
		// 处理当前语句逻辑。
		AccessToken: accessToken,
		// 处理当前语句逻辑。
		RefreshToken: refreshToken,
		// 调用buildUserProfile完成当前处理。
		User: buildUserProfile(user),
		// 处理当前语句逻辑。
	}, nil
}

// findUserByPhone 按手机号查询用户。
func (r *Repository) findUserByPhone(ctx context.Context, phone string) (*models.WUser, error) {
	// 更新当前变量或字段值。
	phone = strings.TrimSpace(phone)
	// 判断条件并进入对应分支逻辑。
	if !phoneRegexp.MatchString(phone) {
		// 返回当前处理结果。
		return nil, errInvalidPhone
	}
	// 定义并初始化当前变量。
	row := models.WUser{}
	// 判断条件并进入对应分支逻辑。
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&row).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}
	// 返回当前处理结果。
	return &row, nil
}

// buildUserProfile 统一组装用户资料。
func buildUserProfile(user models.WUser) map[string]interface{} {
	// 返回当前处理结果。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"id": user.ID,
		// 处理当前语句逻辑。
		"username": user.Username,
		// 处理当前语句逻辑。
		"phone": user.Phone,
		// 处理当前语句逻辑。
		"nickname": user.Nickname,
		// 处理当前语句逻辑。
		"avatar": user.Avatar,
		// 处理当前语句逻辑。
		"user_type": user.UserType,
		// 处理当前语句逻辑。
		"status": user.Status,
		// 处理当前语句逻辑。
		"register_source": user.RegisterSource,
		// 处理当前语句逻辑。
		"last_login_at": user.LastLoginAt,
	}
}

// generateSMSCode 使用密码学安全随机源生成 6 位验证码。
func generateSMSCode() (string, error) {
	// 定义并初始化当前变量。
	upper := big.NewInt(1000000)
	// 定义并初始化当前变量。
	n, err := rand.Int(rand.Reader, upper)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return "", err
	}
	// 返回当前处理结果。
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// exposeSMSPreviewCode 控制是否允许返回测试验证码。
func exposeSMSPreviewCode() bool {
	// 定义并初始化当前变量。
	flag := strings.ToLower(strings.TrimSpace(os.Getenv("TK_USER_EXPOSE_SMS_PREVIEW")))
	// 判断条件并进入对应分支逻辑。
	return flag == "1" || flag == "true" || flag == "yes" || flag == "on"
}

// right 返回字符串右侧 n 位字符。
func right(raw string, n int) string {
	// 判断条件并进入对应分支逻辑。
	if n <= 0 {
		// 返回当前处理结果。
		return ""
	}
	// 判断条件并进入对应分支逻辑。
	if len(raw) <= n {
		// 返回当前处理结果。
		return raw
	}
	// 返回当前处理结果。
	return raw[len(raw)-n:]
}
