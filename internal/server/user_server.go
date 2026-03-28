package server

import (
	"context"
	"encoding/json"

	tkv1 "github.com/wangyahua6688-maker/tk-proto/gen/go/tk/v1"
	"tk-user/internal/svc"
)

// UserServer 用户域 gRPC 服务。
// 当前职责：
// 1) 提供论坛帖子列表；
// 2) 提供彩种详情评论分组；
// 3) 未来可扩展登录、资料、关系链等用户能力。
type UserServer struct {
	// 处理当前语句逻辑。
	tkv1.UnimplementedUserServiceServer
	authRPC  *AuthRPC
	forumRPC *ForumRPC
}

// NewUserServer 创建用户域服务实例。
func NewUserServer(ctx *svc.ServiceContext) *UserServer {
	return &UserServer{
		authRPC:  NewAuthRPC(ctx),
		forumRPC: NewForumRPC(ctx),
	}
}

// TopicList 转发到论坛模块 RPC。
func (s *UserServer) TopicList(ctx context.Context, req *tkv1.TopicListRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.TopicList(ctx, req)
}

// ForumTopics 转发到论坛模块 RPC。
func (s *UserServer) ForumTopics(ctx context.Context, req *tkv1.ForumTopicsRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.ForumTopics(ctx, req)
}

// ForumTopicDetail 转发到论坛模块 RPC。
func (s *UserServer) ForumTopicDetail(ctx context.Context, req *tkv1.ForumTopicDetailRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.ForumTopicDetail(ctx, req)
}

// ForumAuthorHistory 转发到论坛模块 RPC。
func (s *UserServer) ForumAuthorHistory(ctx context.Context, req *tkv1.ForumAuthorHistoryRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.ForumAuthorHistory(ctx, req)
}

// ExpertBoards 转发到论坛模块 RPC。
func (s *UserServer) ExpertBoards(ctx context.Context, req *tkv1.ExpertBoardsRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.ExpertBoards(ctx, req)
}

// SendSMSCode 转发到鉴权模块 RPC。
func (s *UserServer) SendSMSCode(ctx context.Context, req *tkv1.AuthSendCodeRequest) (*tkv1.JsonDataReply, error) {
	return s.authRPC.SendSMSCode(ctx, req)
}

// RegisterByPhone 转发到鉴权模块 RPC。
func (s *UserServer) RegisterByPhone(ctx context.Context, req *tkv1.AuthRegisterRequest) (*tkv1.JsonDataReply, error) {
	return s.authRPC.RegisterByPhone(ctx, req)
}

// LoginByPassword 转发到鉴权模块 RPC。
func (s *UserServer) LoginByPassword(ctx context.Context, req *tkv1.AuthPasswordLoginRequest) (*tkv1.JsonDataReply, error) {
	return s.authRPC.LoginByPassword(ctx, req)
}

// LoginBySMS 转发到鉴权模块 RPC。
func (s *UserServer) LoginBySMS(ctx context.Context, req *tkv1.AuthSMSLoginRequest) (*tkv1.JsonDataReply, error) {
	return s.authRPC.LoginBySMS(ctx, req)
}

// Profile 转发到鉴权模块 RPC。
func (s *UserServer) Profile(ctx context.Context, req *tkv1.AuthProfileRequest) (*tkv1.JsonDataReply, error) {
	return s.authRPC.Profile(ctx, req)
}

// LotteryCommentGroups 转发到论坛模块 RPC。
func (s *UserServer) LotteryCommentGroups(ctx context.Context, req *tkv1.LotteryCommentGroupsRequest) (*tkv1.JsonDataReply, error) {
	return s.forumRPC.LotteryCommentGroups(ctx, req)
}

// marshalOK 将任意 payload 转为 JsonDataReply。
func marshalOK(payload interface{}) (*tkv1.JsonDataReply, error) {
	// 定义并初始化当前变量。
	raw, err := json.Marshal(payload)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50099, Msg: "marshal response failed"}, nil
	}
	// 返回当前处理结果。
	return &tkv1.JsonDataReply{Code: 0, Msg: "ok", DataJson: string(raw)}, nil
}
