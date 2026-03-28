package repo

import "context"

// ListTopics 返回论坛帖子列表，并通过聚合查询避免逐条统计评论数。
func (r *ForumRepository) ListTopics(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	// 旧接口复用新论坛查询逻辑，保持兼容。
	result, err := r.ListForumTopics(ctx, ForumTopicQuery{
		// 处理当前语句逻辑。
		Limit: limit,
		// 处理当前语句逻辑。
		Feed: "all",
	})
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}
	// 返回当前处理结果。
	return result.Items, nil
}
