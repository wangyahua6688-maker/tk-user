package server

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	tkv1 "tk-proto/tk/v1"
	"tk-user/internal/repo"
)

// TopicList 返回论坛帖子列表（旧兼容接口）。
func (s *UserServer) TopicList(ctx context.Context, req *tkv1.TopicListRequest) (*tkv1.JsonDataReply, error) {
	// 1) 旧接口默认走 all 分栏。
	limit := int(req.GetLimit())
	// 定义并初始化当前变量。
	items, err := s.ctx.CommentRepo.ListTopics(ctx, limit)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50021, Msg: "failed to load topics"}, nil
	}
	// 2) 返回兼容结构。
	return marshalOK(map[string]interface{}{
		// 处理当前语句逻辑。
		"feed": "all",
		// 处理当前语句逻辑。
		"items": items,
		// 调用len完成当前处理。
		"total": len(items),
	})
}

// ForumTopics 返回论坛列表（新接口，支持 feed/keyword）。
func (s *UserServer) ForumTopics(ctx context.Context, req *tkv1.ForumTopicsRequest) (*tkv1.JsonDataReply, error) {
	// 1) 读取分页与筛选参数。
	query := repo.ForumTopicQuery{
		// 调用int完成当前处理。
		Limit: int(req.GetLimit()),
		// 调用strings.TrimSpace完成当前处理。
		Feed: strings.TrimSpace(req.GetFeed()),
		// 调用strings.TrimSpace完成当前处理。
		Keyword: strings.TrimSpace(req.GetKeyword()),
		// 调用strings.TrimSpace完成当前处理。
		Issue: strings.TrimSpace(req.GetIssue()),
		// 调用int完成当前处理。
		Year: int(req.GetYear()),
	}
	// 2) 执行论坛查询。
	result, err := s.ctx.CommentRepo.ListForumTopics(ctx, query)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50023, Msg: "failed to load forum topics"}, nil
	}
	// 3) 输出论坛页聚合结构。
	return marshalOK(map[string]interface{}{
		// 调用normalizeFeed完成当前处理。
		"feed": normalizeFeed(query.Feed),
		// 处理当前语句逻辑。
		"keyword": query.Keyword,
		// 处理当前语句逻辑。
		"issue": query.Issue,
		// 处理当前语句逻辑。
		"year": query.Year,
		// 处理当前语句逻辑。
		"items": result.Items,
		// 处理当前语句逻辑。
		"total": result.Total,
		// 进入新的代码块进行处理。
		"tabs": []map[string]interface{}{
			// 处理当前语句逻辑。
			{"key": "all", "label": "全部"},
			// 处理当前语句逻辑。
			{"key": "latest", "label": "最新贴"},
			// 处理当前语句逻辑。
			{"key": "history", "label": "历史贴"},
		},
		// 处理当前语句逻辑。
		"history_filters": result.HistoryFilters,
	})
}

// ForumTopicDetail 返回论坛帖子详情（含开奖块、作者统计、评论分组）。
func (s *UserServer) ForumTopicDetail(ctx context.Context, req *tkv1.ForumTopicDetailRequest) (*tkv1.JsonDataReply, error) {
	// 1) 帖子ID 必填。
	postID := uint(req.GetPostId())
	// 判断条件并进入对应分支逻辑。
	if postID == 0 {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40031, Msg: "invalid post id"}, nil
	}
	// 2) 查询详情聚合数据。
	payload, err := s.ctx.CommentRepo.ForumTopicDetail(ctx, postID)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 判断条件并进入对应分支逻辑。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回当前处理结果。
			return &tkv1.JsonDataReply{Code: 40431, Msg: "post not found"}, nil
		}
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50031, Msg: "failed to load topic detail"}, nil
	}
	// 3) 返回详情数据。
	return marshalOK(payload)
}

// ForumAuthorHistory 返回作者历史发帖列表。
func (s *UserServer) ForumAuthorHistory(ctx context.Context, req *tkv1.ForumAuthorHistoryRequest) (*tkv1.JsonDataReply, error) {
	// 1) 校验用户ID。
	userID := uint(req.GetUserId())
	// 判断条件并进入对应分支逻辑。
	if userID == 0 {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40032, Msg: "invalid user id"}, nil
	}
	// 2) 查询作者历史贴列表。
	items, err := s.ctx.CommentRepo.ListForumAuthorHistory(
		// 处理当前语句逻辑。
		ctx,
		// 处理当前语句逻辑。
		userID,
		// 调用int完成当前处理。
		int(req.GetLimit()),
		// 调用req.GetIssue完成当前处理。
		req.GetIssue(),
		// 调用int完成当前处理。
		int(req.GetYear()),
	)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50032, Msg: "failed to load author history"}, nil
	}
	// 3) 返回统一结构。
	return marshalOK(map[string]interface{}{
		// 处理当前语句逻辑。
		"user_id": userID,
		// 调用strings.TrimSpace完成当前处理。
		"issue": strings.TrimSpace(req.GetIssue()),
		// 调用int完成当前处理。
		"year": int(req.GetYear()),
		// 处理当前语句逻辑。
		"items": items,
		// 调用len完成当前处理。
		"total": len(items),
	})
}

// ExpertBoards 返回高手推荐榜单。
func (s *UserServer) ExpertBoards(ctx context.Context, req *tkv1.ExpertBoardsRequest) (*tkv1.JsonDataReply, error) {
	// 1) 拉取榜单分组数据。
	payload, err := s.ctx.CommentRepo.ListExpertBoards(ctx, int(req.GetLimit()), req.GetLotteryCode())
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50024, Msg: "failed to load expert boards"}, nil
	}
	// 2) 输出榜单结果。
	return marshalOK(payload)
}

// LotteryCommentGroups 返回彩种详情页评论分组。
func (s *UserServer) LotteryCommentGroups(ctx context.Context, req *tkv1.LotteryCommentGroupsRequest) (*tkv1.JsonDataReply, error) {
	// 1) lottery_info_id 必填。
	if req.GetLotteryInfoId() == 0 {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 40012, Msg: "invalid lottery info id"}, nil
	}
	// 2) 查询系统/网友/热门/最新评论四组数据。
	payload, err := s.ctx.CommentRepo.LotteryCommentGroups(ctx, uint(req.GetLotteryInfoId()))
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return &tkv1.JsonDataReply{Code: 50022, Msg: "failed to load comments"}, nil
	}
	// 3) 返回评论分组结构。
	return marshalOK(payload)
}

// normalizeFeed 标准化 feed 枚举值。
func normalizeFeed(feed string) string {
	// 根据表达式进入多分支处理。
	switch strings.ToLower(strings.TrimSpace(feed)) {
	case "latest":
		return "latest"
	case "history":
		return "history"
	default:
		return "all"
	}
}
