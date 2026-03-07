package repo

import "context"

// ListTopics 返回论坛帖子列表，并通过聚合查询避免逐条统计评论数。
func (r *Repository) ListTopics(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// 旧接口复用新论坛查询逻辑，保持兼容。
	return r.ListForumTopics(ctx, limit, "all", "")
}
