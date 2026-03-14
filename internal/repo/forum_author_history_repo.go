package repo

import (
	"context"
	"strings"
)

// ListForumAuthorHistory 查询指定用户的历史发帖列表。
func (r *Repository) ListForumAuthorHistory(
	ctx context.Context,
	userID uint,
	limit int,
	issue string,
	year int,
) ([]map[string]interface{}, error) {
	// 1) 参数保护：userID 必填。
	if userID == 0 {
		return []map[string]interface{}{}, nil
	}
	// 2) 限制最大条数，防止超大分页压垮数据库。
	if limit <= 0 {
		limit = 20
	}
	if limit > 120 {
		limit = 120
	}

	// 3) 统一清洗期号参数。
	issue = strings.TrimSpace(issue)

	// 4) 查询帖子 + 用户 + 互动统计 + 关联期号信息。
	rows := make([]topicRow, 0)
	q := r.db.WithContext(ctx).
		Table("tk_post_article AS p").
		Select(`p.id, p.user_id, p.lottery_info_id, p.title, p.content, p.cover_image, p.is_official, p.created_at,
				COALESCE(cc.comment_count, 0) AS comment_count,
				COALESCE(lk.like_count, 0) AS like_count,
				COALESCE(li.issue, '') AS issue,
				COALESCE(li.year, 0) AS year,
				COALESCE(li.special_lottery_id, 0) AS special_lottery_id,
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
		Joins("LEFT JOIN tk_lottery_info AS li ON li.id = p.lottery_info_id").
		Joins("LEFT JOIN tk_users AS u ON u.id = p.user_id").
		Where("p.status = 1 AND p.user_id = ?", userID)

	// 5) 期号/年份筛选（历史发帖页使用）。
	if issue != "" {
		q = q.Where("li.issue = ?", issue)
	}
	if year > 0 {
		q = q.Where("li.year = ?", year)
	}

	// 6) 执行查询并按时间倒序。
	if err := q.Order("p.created_at DESC, p.id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// 7) 组装输出结构。
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]interface{}{
			"id":                 row.ID,
			"user_id":            row.UserID,
			"lottery_info_id":    row.LotteryInfoID,
			"title":              row.Title,
			"cover_image":        row.CoverImage,
			"content_preview":    trimTextPreview(row.Content, 120),
			"is_official":        row.IsOfficial == 1,
			"comment_count":      row.CommentCount,
			"like_count":         row.LikeCount,
			"issue":              row.Issue,
			"year":               row.Year,
			"special_lottery_id": row.SpecialLotteryID,
			"created_at":         row.CreatedAt,
			"user": map[string]interface{}{
				"id":        row.UserID,
				"username":  row.Username,
				"nickname":  row.Nickname,
				"avatar":    row.Avatar,
				"user_type": row.UserType,
			},
		})
	}
	return items, nil
}
