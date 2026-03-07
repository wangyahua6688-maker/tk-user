package repo

import (
	"context"
	"fmt"
	"strings"
)

// ListForumTopics 返回论坛帖子列表（支持分栏与关键字筛选）。
func (r *Repository) ListForumTopics(ctx context.Context, limit int, feed string, keyword string) ([]map[string]interface{}, error) {
	// 1) 限制返回上限，防止一次性拉取过大数据。
	if limit <= 0 {
		limit = 20
	}
	if limit > 120 {
		limit = 120
	}

	// 2) 标准化参数，便于缓存命中。
	feedKey := strings.ToLower(strings.TrimSpace(feed))
	if feedKey == "" {
		feedKey = "all"
	}
	keyword = strings.TrimSpace(keyword)

	// 3) 尝试读取缓存，降低论坛高并发下的数据库压力。
	cacheKey := fmt.Sprintf("tk:forum:topics:%s:%s:%d", feedKey, strings.ToLower(keyword), limit)
	cacheHit := make([]map[string]interface{}, 0)
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		return cacheHit, nil
	}

	// 4) 构建主查询，聚合评论数量和互动热度。
	rows := make([]topicRow, 0)
	q := r.db.WithContext(ctx).
		Table("tk_post_article AS p").
		Select(`p.id, p.user_id, p.title, p.cover_image, p.is_official, p.created_at,
				COALESCE(cc.comment_count, 0) AS comment_count,
				COALESCE(lk.like_count, 0) AS like_count,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		Joins(`LEFT JOIN (
				SELECT post_id, COUNT(1) AS comment_count
				FROM tk_comment
				WHERE status = 1
				GROUP BY post_id
			) AS cc ON cc.post_id = p.id`).
		Joins(`LEFT JOIN (
				SELECT post_id, COALESCE(SUM(likes), 0) AS like_count
				FROM tk_comment
				WHERE status = 1
				GROUP BY post_id
			) AS lk ON lk.post_id = p.id`).
		Joins("LEFT JOIN tk_users AS u ON u.id = p.user_id").
		Where("p.status = 1")

	// 5) 关键字筛选：标题优先，正文兜底。
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("p.title LIKE ? OR p.content LIKE ?", like, like)
	}

	// 6) 按分栏控制排序策略。
	switch feedKey {
	case "latest":
		q = q.Order("p.created_at DESC, p.id DESC")
	case "history":
		q = q.Order("p.created_at ASC, p.id ASC")
	default:
		q = q.Order("p.is_official DESC, (COALESCE(cc.comment_count,0) + COALESCE(lk.like_count,0)) DESC, p.id DESC")
	}

	// 7) 执行查询。
	if err := q.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// 8) 转换为对外 JSON 结构。
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]interface{}{
			"id":            row.ID,
			"user_id":       row.UserID,
			"title":         row.Title,
			"is_official":   row.IsOfficial == 1,
			"cover_image":   row.CoverImage,
			"comment_count": row.CommentCount,
			"like_count":    row.LikeCount,
			"created_at":    row.CreatedAt,
			"user": map[string]interface{}{
				"id":       row.UserID,
				"username": row.Username,
				"nickname": row.Nickname,
				"avatar":   row.Avatar,
				"user_type": row.UserType,
			},
		})
	}

	// 9) 写缓存，保障后续同参数请求快速返回。
	r.saveCache(ctx, cacheKey, items)
	return items, nil
}

