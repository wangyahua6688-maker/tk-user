package repo

import (
	"context"
	"strings"
)

// ListForumAuthorHistory 查询指定用户的历史发帖列表。
func (r *ForumRepository) ListForumAuthorHistory(
	// 处理当前语句逻辑。
	ctx context.Context,
	// 处理当前语句逻辑。
	userID uint,
	// 处理当前语句逻辑。
	limit int,
	// 处理当前语句逻辑。
	issue string,
	// 处理当前语句逻辑。
	year int,
	// 进入新的代码块进行处理。
) ([]map[string]interface{}, error) {
	// 1) 参数保护：userID 必填。
	if userID == 0 {
		// 返回当前处理结果。
		return []map[string]interface{}{}, nil
	}
	// 2) 限制最大条数，防止超大分页压垮数据库。
	if limit <= 0 {
		// 更新当前变量或字段值。
		limit = 20
	}
	// 判断条件并进入对应分支逻辑。
	if limit > 120 {
		// 更新当前变量或字段值。
		limit = 120
	}

	// 3) 统一清洗期号参数。
	issue = strings.TrimSpace(issue)

	// 4) 查询帖子 + 用户 + 互动统计 + 关联期号信息。
	rows := make([]topicRow, 0)
	// 定义并初始化当前变量。
	q := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_post_article AS p").
		// 调用Select完成当前处理。
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
		// 调用Joins完成当前处理。
		Joins(`LEFT JOIN (
				SELECT post_id, COUNT(1) AS comment_count
				FROM tk_comment
				WHERE status = 1
				GROUP BY post_id
			) AS cc ON cc.post_id = p.id`).
		// 调用Joins完成当前处理。
		Joins(`LEFT JOIN (
				SELECT post_id, COALESCE(SUM(likes), 0) AS like_count
				FROM tk_comment
				WHERE status = 1
				GROUP BY post_id
			) AS lk ON lk.post_id = p.id`).
		// 更新当前变量或字段值。
		Joins("LEFT JOIN tk_lottery_info AS li ON li.id = p.lottery_info_id").
		// 更新当前变量或字段值。
		Joins("LEFT JOIN tk_users AS u ON u.id = p.user_id").
		// 更新当前变量或字段值。
		Where("p.status = 1 AND p.user_id = ?", userID)

	// 5) 期号/年份筛选（历史发帖页使用）。
	if issue != "" {
		// 更新当前变量或字段值。
		q = q.Where("li.issue = ?", issue)
	}
	// 判断条件并进入对应分支逻辑。
	if year > 0 {
		// 更新当前变量或字段值。
		q = q.Where("li.year = ?", year)
	}

	// 6) 执行查询并按时间倒序。
	if err := q.Order("p.created_at DESC, p.id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 7) 组装输出结构。
	items := make([]map[string]interface{}, 0, len(rows))
	// 循环处理当前数据集合。
	for _, row := range rows {
		// 更新当前变量或字段值。
		items = append(items, map[string]interface{}{
			// 处理当前语句逻辑。
			"id": row.ID,
			// 处理当前语句逻辑。
			"user_id": row.UserID,
			// 处理当前语句逻辑。
			"lottery_info_id": row.LotteryInfoID,
			// 处理当前语句逻辑。
			"title": row.Title,
			// 处理当前语句逻辑。
			"cover_image": row.CoverImage,
			// 调用trimTextPreview完成当前处理。
			"content_preview": trimTextPreview(row.Content, 120),
			// 处理当前语句逻辑。
			"is_official": row.IsOfficial == 1,
			// 处理当前语句逻辑。
			"comment_count": row.CommentCount,
			// 处理当前语句逻辑。
			"like_count": row.LikeCount,
			// 处理当前语句逻辑。
			"issue": row.Issue,
			// 处理当前语句逻辑。
			"year": row.Year,
			// 处理当前语句逻辑。
			"special_lottery_id": row.SpecialLotteryID,
			// 处理当前语句逻辑。
			"created_at": row.CreatedAt,
			// 进入新的代码块进行处理。
			"user": map[string]interface{}{
				// 处理当前语句逻辑。
				"id": row.UserID,
				// 处理当前语句逻辑。
				"username": row.Username,
				// 处理当前语句逻辑。
				"nickname": row.Nickname,
				// 处理当前语句逻辑。
				"avatar": row.Avatar,
				// 处理当前语句逻辑。
				"user_type": row.UserType,
			},
		})
	}
	// 返回当前处理结果。
	return items, nil
}
