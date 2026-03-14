package repo

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"tk-common/models"
)

// ForumTopicDetail 查询帖子详情（包含开奖块、作者信息、评论分组、历史贴）。
func (r *Repository) ForumTopicDetail(ctx context.Context, postID uint) (map[string]interface{}, error) {
	// 1) postID 必须有效。
	if postID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	// 2) 查询帖子主体 + 用户 + 评论聚合 + 图纸期号信息。
	row := topicRow{}
	err := r.db.WithContext(ctx).
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
		Where("p.id = ? AND p.status = 1", postID).
		First(&row).Error
	if err != nil {
		return nil, err
	}

	// 3) 解析帖子顶部关联开奖块。
	drawPayload, err := r.resolveForumDrawPayload(ctx, row)
	if err != nil {
		return nil, err
	}

	// 4) 计算作者统计信息（发帖、粉丝、关注、成长值、阅读贴数、获赞）。
	authorStats, err := r.loadForumAuthorStats(ctx, row.UserID)
	if err != nil {
		return nil, err
	}

	// 5) 拉取作者历史贴列表（详情页右上角“历史贴子”使用）。
	authorHistory, err := r.ListForumAuthorHistory(ctx, row.UserID, 30, "", 0)
	if err != nil {
		return nil, err
	}

	// 6) 查询热门评论与最新评论（一级评论 + 预览回复）。
	hotComments, err := r.listForumPostComments(ctx, postID, "hot", 20)
	if err != nil {
		return nil, err
	}
	latestComments, err := r.listForumPostComments(ctx, postID, "latest", 20)
	if err != nil {
		return nil, err
	}

	// 7) 组装详情页聚合响应结构。
	return map[string]interface{}{
		"topic":           buildForumTopicPayload(row),
		"draw":            drawPayload,
		"author":          buildForumAuthorPayload(row, authorStats),
		"author_history":  authorHistory,
		"hot_comments":    hotComments,
		"latest_comments": latestComments,
		"comment_total":   row.CommentCount,
	}, nil
}

// loadForumAuthorStats 聚合作者统计数据。
func (r *Repository) loadForumAuthorStats(ctx context.Context, userID uint) (map[string]interface{}, error) {
	// 1) user_id 无效时返回零值结构。
	if userID == 0 {
		return map[string]interface{}{
			"post_count":      0,
			"fans_count":      0,
			"following_count": 0,
			"growth_value":    0,
			"read_post_count": 0,
			"liked_count":     0,
		}, nil
	}

	// 2) 读取发帖数量。
	var postCount int64
	if err := r.db.WithContext(ctx).
		Table("tk_post_article").
		Where("status = 1 AND user_id = ?", userID).
		Count(&postCount).Error; err != nil {
		return nil, err
	}

	// 3) 聚合“帖子获赞总数”（按该作者帖子下评论 likes 求和）。
	var likedCount int64
	if err := r.db.WithContext(ctx).
		Table("tk_post_article AS p").
		Select("COALESCE(SUM(c.likes), 0)").
		Joins("LEFT JOIN tk_comment AS c ON c.post_id = p.id AND c.status = 1").
		Where("p.status = 1 AND p.user_id = ?", userID).
		Scan(&likedCount).Error; err != nil {
		return nil, err
	}

	// 4) 粉丝/关注/成长值/阅读贴数来源于用户扩展字段（存在则读取，不存在则回退 0）。
	fansCount, followingCount, growthValue, readPostCount := int64(0), int64(0), int64(0), int64(0)
	if r.db.Migrator().HasColumn(&models.WUser{}, "fans_count") &&
		r.db.Migrator().HasColumn(&models.WUser{}, "following_count") &&
		r.db.Migrator().HasColumn(&models.WUser{}, "growth_value") &&
		r.db.Migrator().HasColumn(&models.WUser{}, "read_post_count") {
		type metricsRow struct {
			FansCount      int64 `json:"fans_count"`
			FollowingCount int64 `json:"following_count"`
			GrowthValue    int64 `json:"growth_value"`
			ReadPostCount  int64 `json:"read_post_count"`
		}
		mr := metricsRow{}
		if err := r.db.WithContext(ctx).
			Table("tk_users").
			Select("fans_count, following_count, growth_value, read_post_count").
			Where("id = ?", userID).
			First(&mr).Error; err == nil {
			fansCount = mr.FansCount
			followingCount = mr.FollowingCount
			growthValue = mr.GrowthValue
			readPostCount = mr.ReadPostCount
		}
	}

	// 5) 返回统一统计结构。
	return map[string]interface{}{
		"post_count":      postCount,
		"fans_count":      fansCount,
		"following_count": followingCount,
		"growth_value":    growthValue,
		"read_post_count": readPostCount,
		"liked_count":     likedCount,
	}, nil
}

// listForumPostComments 查询帖子评论列表（含回复预览）。
func (r *Repository) listForumPostComments(
	ctx context.Context,
	postID uint,
	orderBy string,
	limit int,
) ([]map[string]interface{}, error) {
	// 1) 默认条数兜底。
	if limit <= 0 {
		limit = 10
	}
	if limit > 80 {
		limit = 80
	}

	// 2) 查询一级评论 + 作者信息 + 回复数。
	rows := make([]commentRow, 0)
	q := r.db.WithContext(ctx).
		Table("tk_comment AS c").
		Select(`c.id, c.user_id, c.parent_id, c.content, c.likes, c.created_at,
				COALESCE(rc.reply_count, 0) AS reply_count,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		Joins(`LEFT JOIN (
				SELECT parent_id, COUNT(1) AS reply_count
				FROM tk_comment
				WHERE status = 1 AND post_id = ? AND parent_id > 0
				GROUP BY parent_id
			) AS rc ON rc.parent_id = c.id`, postID).
		Joins("LEFT JOIN tk_users AS u ON u.id = c.user_id").
		Where("c.status = 1 AND c.post_id = ? AND c.parent_id = 0", postID)

	// 3) 热门按点赞排序，默认按时间倒序。
	if strings.EqualFold(strings.TrimSpace(orderBy), "hot") {
		q = q.Order("c.likes DESC, c.id DESC")
	} else {
		q = q.Order("c.created_at DESC, c.id DESC")
	}

	// 4) 执行查询。
	if err := q.Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	// 5) 组装评论 + 回复预览。
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		replies, err := r.listForumCommentReplies(ctx, postID, row.ID, 3)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id":          row.ID,
			"user_id":     row.UserID,
			"content":     row.Content,
			"likes":       row.Likes,
			"reply_count": row.ReplyCount,
			"created_at":  row.CreatedAt,
			"user": map[string]interface{}{
				"id":        row.UserID,
				"username":  row.Username,
				"nickname":  row.Nickname,
				"avatar":    row.Avatar,
				"user_type": row.UserType,
			},
			"replies": replies,
		})
	}
	return out, nil
}

// listForumCommentReplies 查询指定一级评论下的回复预览。
func (r *Repository) listForumCommentReplies(
	ctx context.Context,
	postID uint,
	parentID uint,
	limit int,
) ([]map[string]interface{}, error) {
	// 1) 参数保护。
	if parentID == 0 || limit <= 0 {
		return []map[string]interface{}{}, nil
	}
	if limit > 20 {
		limit = 20
	}

	// 2) 查询回复。
	rows := make([]commentRow, 0)
	err := r.db.WithContext(ctx).
		Table("tk_comment AS c").
		Select(`c.id, c.user_id, c.parent_id, c.content, c.likes, c.created_at,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		Joins("LEFT JOIN tk_users AS u ON u.id = c.user_id").
		Where("c.status = 1 AND c.post_id = ? AND c.parent_id = ?", postID, parentID).
		Order("c.created_at ASC, c.id ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 3) 转换响应结构。
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]interface{}{
			"id":         row.ID,
			"user_id":    row.UserID,
			"parent_id":  row.ParentID,
			"content":    row.Content,
			"likes":      row.Likes,
			"created_at": row.CreatedAt,
			"user": map[string]interface{}{
				"id":        row.UserID,
				"username":  row.Username,
				"nickname":  row.Nickname,
				"avatar":    row.Avatar,
				"user_type": row.UserType,
			},
		})
	}
	return out, nil
}

// buildForumTopicPayload 组装帖子主信息。
func buildForumTopicPayload(row topicRow) map[string]interface{} {
	// 1) 用户显示名兜底优先级：昵称 -> 用户名 -> 用户ID。
	displayName := strings.TrimSpace(row.Nickname)
	if displayName == "" {
		displayName = strings.TrimSpace(row.Username)
	}
	if displayName == "" {
		displayName = "用户"
	}

	// 2) 返回前端可直接消费的结构。
	return map[string]interface{}{
		"id":                 row.ID,
		"user_id":            row.UserID,
		"lottery_info_id":    row.LotteryInfoID,
		"title":              row.Title,
		"content":            row.Content,
		"cover_image":        row.CoverImage,
		"is_official":        row.IsOfficial == 1,
		"comment_count":      row.CommentCount,
		"like_count":         row.LikeCount,
		"issue":              row.Issue,
		"year":               row.Year,
		"special_lottery_id": row.SpecialLotteryID,
		"created_at":         row.CreatedAt,
		"user": map[string]interface{}{
			"id":           row.UserID,
			"username":     row.Username,
			"nickname":     row.Nickname,
			"display_name": displayName,
			"avatar":       row.Avatar,
			"user_type":    row.UserType,
		},
	}
}

// buildForumAuthorPayload 组装作者信息与统计字段。
func buildForumAuthorPayload(row topicRow, stats map[string]interface{}) map[string]interface{} {
	// 1) 作者显示名兜底。
	displayName := strings.TrimSpace(row.Nickname)
	if displayName == "" {
		displayName = strings.TrimSpace(row.Username)
	}
	if displayName == "" {
		displayName = "用户"
	}

	// 2) 输出作者结构。
	return map[string]interface{}{
		"id":           row.UserID,
		"username":     row.Username,
		"nickname":     row.Nickname,
		"display_name": displayName,
		"avatar":       row.Avatar,
		"user_type":    row.UserType,
		"stats":        stats,
	}
}

// IsForumDetailNotFound 判断论坛详情是否为记录不存在。
func IsForumDetailNotFound(err error) bool {
	// 允许上层将 gorm 的 not found 转为业务 404。
	return errors.Is(err, gorm.ErrRecordNotFound)
}
