package services

import (
	"context"

	"tk-user/internal/dto"
)

// authRepository 定义认证模块仓储依赖。
type authRepository interface {
	SendSMSCode(ctx context.Context, phone string, purpose string) (dto.SMSResult, error)
	RegisterByPhone(ctx context.Context, phone string, password string, smsCode string, nickname string) (dto.AuthResult, error)
	LoginByPassword(ctx context.Context, phone string, password string) (dto.AuthResult, error)
	LoginBySMS(ctx context.Context, phone string, smsCode string) (dto.AuthResult, error)
	ProfileByToken(ctx context.Context, accessToken string) (map[string]interface{}, error)
}

// AuthService 封装认证模块业务逻辑。
type AuthService struct {
	repo authRepository
}

// NewAuthService 创建认证模块服务。
func NewAuthService(repo authRepository) *AuthService {
	return &AuthService{repo: repo}
}

// SendSMSCode 发送短信验证码。
func (s *AuthService) SendSMSCode(ctx context.Context, phone string, purpose string) (dto.SMSResult, error) {
	return s.repo.SendSMSCode(ctx, phone, purpose)
}

// RegisterByPhone 手机号注册。
func (s *AuthService) RegisterByPhone(ctx context.Context, phone string, password string, smsCode string, nickname string) (dto.AuthResult, error) {
	return s.repo.RegisterByPhone(ctx, phone, password, smsCode, nickname)
}

// LoginByPassword 手机号密码登录。
func (s *AuthService) LoginByPassword(ctx context.Context, phone string, password string) (dto.AuthResult, error) {
	return s.repo.LoginByPassword(ctx, phone, password)
}

// LoginBySMS 手机号验证码登录。
func (s *AuthService) LoginBySMS(ctx context.Context, phone string, smsCode string) (dto.AuthResult, error) {
	return s.repo.LoginBySMS(ctx, phone, smsCode)
}

// ProfileByToken 根据令牌查询资料。
func (s *AuthService) ProfileByToken(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	return s.repo.ProfileByToken(ctx, accessToken)
}
