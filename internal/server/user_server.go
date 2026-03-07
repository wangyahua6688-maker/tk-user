package server

import (
	"encoding/json"

	tkv1 "tk-proto/tk/v1"
	"tk-user/internal/svc"
)

// UserServer 用户域 gRPC 服务。
// 当前职责：
// 1) 提供论坛帖子列表；
// 2) 提供彩种详情评论分组；
// 3) 未来可扩展登录、资料、关系链等用户能力。
type UserServer struct {
	tkv1.UnimplementedUserServiceServer
	ctx *svc.ServiceContext
}

// NewUserServer 创建用户域服务实例。
func NewUserServer(ctx *svc.ServiceContext) *UserServer {
	return &UserServer{ctx: ctx}
}

// marshalOK 将任意 payload 转为 JsonDataReply。
func marshalOK(payload interface{}) (*tkv1.JsonDataReply, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return &tkv1.JsonDataReply{Code: 50099, Msg: "marshal response failed"}, nil
	}
	return &tkv1.JsonDataReply{Code: 0, Msg: "ok", DataJson: string(raw)}, nil
}
