package server

import (
	"context"
	"strings"

	tkv1 "tk-proto/tk/v1"
)

// SendSMSCode 发送短信验证码（登录/注册）。
func (s *UserServer) SendSMSCode(ctx context.Context, req *tkv1.AuthSendCodeRequest) (*tkv1.JsonDataReply, error) {
	// 1) 调用仓储层执行验证码发送和频控。
	payload, err := s.ctx.CommentRepo.SendSMSCode(ctx, req.GetPhone(), req.GetPurpose())
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40021, Msg: err.Error()}, nil
	}
	// 2) 输出验证码发送结果。
	return marshalOK(payload)
}

// RegisterByPhone 手机号注册。
func (s *UserServer) RegisterByPhone(ctx context.Context, req *tkv1.AuthRegisterRequest) (*tkv1.JsonDataReply, error) {
	// 1) 创建用户并签发登录态。
	payload, err := s.ctx.CommentRepo.RegisterByPhone(
		// 处理当前语句逻辑。
		ctx,
		// 调用req.GetPhone完成当前处理。
		req.GetPhone(),
		// 调用req.GetPassword完成当前处理。
		req.GetPassword(),
		// 调用req.GetSmsCode完成当前处理。
		req.GetSmsCode(),
		// 调用req.GetNickname完成当前处理。
		req.GetNickname(),
	)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40022, Msg: err.Error()}, nil
	}
	// 2) 返回注册登录结果。
	return marshalOK(payload)
}

// LoginByPassword 手机号密码登录。
func (s *UserServer) LoginByPassword(ctx context.Context, req *tkv1.AuthPasswordLoginRequest) (*tkv1.JsonDataReply, error) {
	// 1) 校验并签发 token。
	payload, err := s.ctx.CommentRepo.LoginByPassword(ctx, req.GetPhone(), req.GetPassword())
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40023, Msg: err.Error()}, nil
	}
	// 2) 返回登录结果。
	return marshalOK(payload)
}

// LoginBySMS 手机号验证码登录。
func (s *UserServer) LoginBySMS(ctx context.Context, req *tkv1.AuthSMSLoginRequest) (*tkv1.JsonDataReply, error) {
	// 1) 校验验证码并创建/登录账号。
	payload, err := s.ctx.CommentRepo.LoginBySMS(ctx, req.GetPhone(), req.GetSmsCode())
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40024, Msg: err.Error()}, nil
	}
	// 2) 返回登录结果。
	return marshalOK(payload)
}

// Profile 根据 access token 获取用户资料。
func (s *UserServer) Profile(ctx context.Context, req *tkv1.AuthProfileRequest) (*tkv1.JsonDataReply, error) {
	// 1) token 优先使用 RPC 字段，兼容传入 Bearer 前缀。
	token := strings.TrimSpace(req.GetAccessToken())
	// 更新当前变量或字段值。
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	// 判断条件并进入对应分支逻辑。
	if token == "" {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40025, Msg: "access token required"}, nil
	}

	// 2) 读取资料。
	profile, err := s.ctx.CommentRepo.ProfileByToken(ctx, token)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40101, Msg: err.Error()}, nil
	}
	// 3) 返回资料结构。
	return marshalOK(profile)
}
