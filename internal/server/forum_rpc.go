package server

import (
	"context"
	"strings"

	tkv1 "tk-proto/tk/v1"
)

// TopicList 返回论坛帖子列表（旧兼容接口）。
func (s *UserServer) TopicList(ctx context.Context, req *tkv1.TopicListRequest) (*tkv1.JsonDataReply, error) {
	// 1) 旧接口默认走 all 分栏。
	limit := int(req.GetLimit())
	items, err := s.ctx.CommentRepo.ListTopics(ctx, limit)
	if err != nil {
		return &tkv1.JsonDataReply{Code: 50021, Msg: "failed to load topics"}, nil
	}
	// 2) 返回兼容结构。
	return marshalOK(map[string]interface{}{
		"feed":  "all",
		"items": items,
		"total": len(items),
	})
}

// ForumTopics 返回论坛列表（新接口，支持 feed/keyword）。
func (s *UserServer) ForumTopics(ctx context.Context, req *tkv1.ForumTopicsRequest) (*tkv1.JsonDataReply, error) {
	// 1) 读取分页与筛选参数。
	limit := int(req.GetLimit())
	feed := strings.TrimSpace(req.GetFeed())
	keyword := strings.TrimSpace(req.GetKeyword())
	// 2) 执行论坛查询。
	items, err := s.ctx.CommentRepo.ListForumTopics(ctx, limit, feed, keyword)
	if err != nil {
		return &tkv1.JsonDataReply{Code: 50023, Msg: "failed to load forum topics"}, nil
	}
	// 3) 输出论坛页聚合结构。
	return marshalOK(map[string]interface{}{
		"feed":    normalizeFeed(feed),
		"keyword": keyword,
		"items":   items,
		"total":   len(items),
		"tabs": []map[string]interface{}{
			{"key": "all", "label": "全部"},
			{"key": "latest", "label": "最新贴"},
			{"key": "history", "label": "历史贴"},
		},
	})
}

// ExpertBoards 返回高手推荐榜单。
func (s *UserServer) ExpertBoards(ctx context.Context, req *tkv1.ExpertBoardsRequest) (*tkv1.JsonDataReply, error) {
	// 1) 拉取榜单分组数据。
	payload, err := s.ctx.CommentRepo.ListExpertBoards(ctx, int(req.GetLimit()), req.GetLotteryCode())
	if err != nil {
		return &tkv1.JsonDataReply{Code: 50024, Msg: "failed to load expert boards"}, nil
	}
	// 2) 输出榜单结果。
	return marshalOK(payload)
}

// LotteryCommentGroups 返回彩种详情页评论分组。
func (s *UserServer) LotteryCommentGroups(ctx context.Context, req *tkv1.LotteryCommentGroupsRequest) (*tkv1.JsonDataReply, error) {
	// 1) lottery_info_id 必填。
	if req.GetLotteryInfoId() == 0 {
		return &tkv1.JsonDataReply{Code: 40012, Msg: "invalid lottery info id"}, nil
	}
	// 2) 查询系统/网友/热门/最新评论四组数据。
	payload, err := s.ctx.CommentRepo.LotteryCommentGroups(ctx, uint(req.GetLotteryInfoId()))
	if err != nil {
		return &tkv1.JsonDataReply{Code: 50022, Msg: "failed to load comments"}, nil
	}
	// 3) 返回评论分组结构。
	return marshalOK(payload)
}

// normalizeFeed 标准化 feed 枚举值。
func normalizeFeed(feed string) string {
	switch strings.ToLower(strings.TrimSpace(feed)) {
	case "latest":
		return "latest"
	case "history":
		return "history"
	default:
		return "all"
	}
}

