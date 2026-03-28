package services

import (
	"context"

	"tk-user/internal/dto"
)

// forumRepository 定义论坛模块仓储依赖。
type forumRepository interface {
	ListTopics(ctx context.Context, limit int) ([]map[string]interface{}, error)
	ListForumTopics(ctx context.Context, query dto.ForumTopicQuery) (dto.ForumTopicListResult, error)
	ForumTopicDetail(ctx context.Context, postID uint) (map[string]interface{}, error)
	ListForumAuthorHistory(ctx context.Context, userID uint, limit int, issue string, year int) ([]map[string]interface{}, error)
	ListExpertBoards(ctx context.Context, limit int, lotteryCode string) (map[string]interface{}, error)
	LotteryCommentGroups(ctx context.Context, infoID uint) (dto.LotteryCommentGroups, error)
}

// ForumService 封装论坛模块业务逻辑。
type ForumService struct {
	repo forumRepository
}

// NewForumService 创建论坛模块服务。
func NewForumService(repo forumRepository) *ForumService {
	return &ForumService{repo: repo}
}

// ListTopics 返回兼容旧接口的帖子列表。
func (s *ForumService) ListTopics(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	return s.repo.ListTopics(ctx, limit)
}

// ListForumTopics 返回论坛页帖子列表。
func (s *ForumService) ListForumTopics(ctx context.Context, query dto.ForumTopicQuery) (dto.ForumTopicListResult, error) {
	return s.repo.ListForumTopics(ctx, query)
}

// ForumTopicDetail 返回帖子详情聚合。
func (s *ForumService) ForumTopicDetail(ctx context.Context, postID uint) (map[string]interface{}, error) {
	return s.repo.ForumTopicDetail(ctx, postID)
}

// ListForumAuthorHistory 返回作者历史贴列表。
func (s *ForumService) ListForumAuthorHistory(ctx context.Context, userID uint, limit int, issue string, year int) ([]map[string]interface{}, error) {
	return s.repo.ListForumAuthorHistory(ctx, userID, limit, issue, year)
}

// ListExpertBoards 返回高手榜单。
func (s *ForumService) ListExpertBoards(ctx context.Context, limit int, lotteryCode string) (map[string]interface{}, error) {
	return s.repo.ListExpertBoards(ctx, limit, lotteryCode)
}

// LotteryCommentGroups 返回彩种详情页评论分组。
func (s *ForumService) LotteryCommentGroups(ctx context.Context, infoID uint) (dto.LotteryCommentGroups, error) {
	return s.repo.LotteryCommentGroups(ctx, infoID)
}
