package repo

import (
	"context"
	"strings"

	"github.com/wangyahua6688-maker/tk-common/models"
	"gorm.io/gorm"
)

// ForumTopicDetail 查询帖子详情（包含开奖块、作者信息、评论分组、历史贴）。
func (r *Repository) ForumTopicDetail(ctx context.Context, postID uint) (map[string]interface{}, error) {
	// 1) postID 必须有效。
	if postID == 0 {
		// 返回当前处理结果。
		return nil, gorm.ErrRecordNotFound
	}

	// 2) 查询帖子主体 + 用户 + 评论聚合 + 图纸期号信息。
	row := topicRow{}
	// 定义并初始化当前变量。
	err := r.db.WithContext(ctx).
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
		Where("p.id = ? AND p.status = 1", postID).
		// 调用First完成当前处理。
		First(&row).Error
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 3) 解析帖子顶部关联开奖块。
	drawPayload, err := r.resolveForumDrawPayload(ctx, row)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 4) 计算作者统计信息（发帖、粉丝、关注、成长值、阅读贴数、获赞）。
	authorStats, err := r.loadForumAuthorStats(ctx, row.UserID)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 5) 拉取作者历史贴列表（详情页右上角“历史贴子”使用）。
	authorHistory, err := r.ListForumAuthorHistory(ctx, row.UserID, 30, "", 0)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 6) 查询热门评论与最新评论（一级评论 + 预览回复）。
	hotComments, err := r.listForumPostComments(ctx, postID, "hot", 20)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}
	// 定义并初始化当前变量。
	latestComments, err := r.listForumPostComments(ctx, postID, "latest", 20)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 7) 组装详情页聚合响应结构。
	return map[string]interface{}{
		// 调用buildForumTopicPayload完成当前处理。
		"topic": buildForumTopicPayload(row),
		// 处理当前语句逻辑。
		"draw": drawPayload,
		// 调用buildForumAuthorPayload完成当前处理。
		"author": buildForumAuthorPayload(row, authorStats),
		// 处理当前语句逻辑。
		"author_history": authorHistory,
		// 处理当前语句逻辑。
		"hot_comments": hotComments,
		// 处理当前语句逻辑。
		"latest_comments": latestComments,
		// 处理当前语句逻辑。
		"comment_total": row.CommentCount,
		// 处理当前语句逻辑。
	}, nil
}

// loadForumAuthorStats 聚合作者统计数据。
func (r *Repository) loadForumAuthorStats(ctx context.Context, userID uint) (map[string]interface{}, error) {
	// 1) user_id 无效时返回零值结构。
	if userID == 0 {
		// 返回当前处理结果。
		return map[string]interface{}{
			// 处理当前语句逻辑。
			"post_count": 0,
			// 处理当前语句逻辑。
			"fans_count": 0,
			// 处理当前语句逻辑。
			"following_count": 0,
			// 处理当前语句逻辑。
			"growth_value": 0,
			// 处理当前语句逻辑。
			"read_post_count": 0,
			// 处理当前语句逻辑。
			"liked_count": 0,
			// 处理当前语句逻辑。
		}, nil
	}

	// 2) 读取发帖数量。
	var postCount int64
	// 判断条件并进入对应分支逻辑。
	if err := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_post_article").
		// 更新当前变量或字段值。
		Where("status = 1 AND user_id = ?", userID).
		// 调用Count完成当前处理。
		Count(&postCount).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 3) 聚合“帖子获赞总数”（按该作者帖子下评论 likes 求和）。
	var likedCount int64
	// 判断条件并进入对应分支逻辑。
	if err := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_post_article AS p").
		// 调用Select完成当前处理。
		Select("COALESCE(SUM(c.likes), 0)").
		// 更新当前变量或字段值。
		Joins("LEFT JOIN tk_comment AS c ON c.post_id = p.id AND c.status = 1").
		// 更新当前变量或字段值。
		Where("p.status = 1 AND p.user_id = ?", userID).
		// 调用Scan完成当前处理。
		Scan(&likedCount).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 4) 粉丝/关注/成长值/阅读贴数来源于用户扩展字段（存在则读取，不存在则回退 0）。
	fansCount, followingCount, growthValue, readPostCount := int64(0), int64(0), int64(0), int64(0)
	// 判断条件并进入对应分支逻辑。
	if r.db.Migrator().HasColumn(&models.WUser{}, "fans_count") &&
		// 调用r.db.Migrator完成当前处理。
		r.db.Migrator().HasColumn(&models.WUser{}, "following_count") &&
		// 调用r.db.Migrator完成当前处理。
		r.db.Migrator().HasColumn(&models.WUser{}, "growth_value") &&
		// 调用r.db.Migrator完成当前处理。
		r.db.Migrator().HasColumn(&models.WUser{}, "read_post_count") {
		// 定义当前类型结构。
		type metricsRow struct {
			// 处理当前语句逻辑。
			FansCount int64 `json:"fans_count"`
			// 处理当前语句逻辑。
			FollowingCount int64 `json:"following_count"`
			// 处理当前语句逻辑。
			GrowthValue int64 `json:"growth_value"`
			// 处理当前语句逻辑。
			ReadPostCount int64 `json:"read_post_count"`
		}
		// 定义并初始化当前变量。
		mr := metricsRow{}
		// 判断条件并进入对应分支逻辑。
		if err := r.db.WithContext(ctx).
			// 调用Table完成当前处理。
			Table("tk_users").
			// 调用Select完成当前处理。
			Select("fans_count, following_count, growth_value, read_post_count").
			// 更新当前变量或字段值。
			Where("id = ?", userID).
			// 调用First完成当前处理。
			First(&mr).Error; err == nil {
			// 更新当前变量或字段值。
			fansCount = mr.FansCount
			// 更新当前变量或字段值。
			followingCount = mr.FollowingCount
			// 更新当前变量或字段值。
			growthValue = mr.GrowthValue
			// 更新当前变量或字段值。
			readPostCount = mr.ReadPostCount
		}
	}

	// 5) 返回统一统计结构。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"post_count": postCount,
		// 处理当前语句逻辑。
		"fans_count": fansCount,
		// 处理当前语句逻辑。
		"following_count": followingCount,
		// 处理当前语句逻辑。
		"growth_value": growthValue,
		// 处理当前语句逻辑。
		"read_post_count": readPostCount,
		// 处理当前语句逻辑。
		"liked_count": likedCount,
		// 处理当前语句逻辑。
	}, nil
}

// listForumPostComments 查询帖子评论列表（含回复预览）。
func (r *Repository) listForumPostComments(
	// 处理当前语句逻辑。
	ctx context.Context,
	// 处理当前语句逻辑。
	postID uint,
	// 处理当前语句逻辑。
	orderBy string,
	// 处理当前语句逻辑。
	limit int,
	// 进入新的代码块进行处理。
) ([]map[string]interface{}, error) {
	// 1) 默认条数兜底。
	if limit <= 0 {
		// 更新当前变量或字段值。
		limit = 10
	}
	// 判断条件并进入对应分支逻辑。
	if limit > 80 {
		// 更新当前变量或字段值。
		limit = 80
	}

	// 2) 查询一级评论 + 作者信息 + 回复数。
	rows := make([]commentRow, 0)
	// 定义并初始化当前变量。
	q := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_comment AS c").
		// 调用Select完成当前处理。
		Select(`c.id, c.user_id, c.parent_id, c.content, c.likes, c.created_at,
				COALESCE(rc.reply_count, 0) AS reply_count,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		// 调用Joins完成当前处理。
		Joins(`LEFT JOIN (
				SELECT parent_id, COUNT(1) AS reply_count
				FROM tk_comment
				WHERE status = 1 AND post_id = ? AND parent_id > 0
				GROUP BY parent_id
			) AS rc ON rc.parent_id = c.id`, postID).
		// 更新当前变量或字段值。
		Joins("LEFT JOIN tk_users AS u ON u.id = c.user_id").
		// 更新当前变量或字段值。
		Where("c.status = 1 AND c.post_id = ? AND c.parent_id = 0", postID)

	// 3) 热门按点赞排序，默认按时间倒序。
	if strings.EqualFold(strings.TrimSpace(orderBy), "hot") {
		// 更新当前变量或字段值。
		q = q.Order("c.likes DESC, c.id DESC")
		// 进入新的代码块进行处理。
	} else {
		// 更新当前变量或字段值。
		q = q.Order("c.created_at DESC, c.id DESC")
	}

	// 4) 执行查询。
	if err := q.Limit(limit).Scan(&rows).Error; err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 5) 组装评论 + 回复预览。
	out := make([]map[string]interface{}, 0, len(rows))
	// 循环处理当前数据集合。
	for _, row := range rows {
		// 定义并初始化当前变量。
		replies, err := r.listForumCommentReplies(ctx, postID, row.ID, 3)
		// 判断条件并进入对应分支逻辑。
		if err != nil {
			// 返回当前处理结果。
			return nil, err
		}
		// 更新当前变量或字段值。
		out = append(out, map[string]interface{}{
			// 处理当前语句逻辑。
			"id": row.ID,
			// 处理当前语句逻辑。
			"user_id": row.UserID,
			// 处理当前语句逻辑。
			"content": row.Content,
			// 处理当前语句逻辑。
			"likes": row.Likes,
			// 处理当前语句逻辑。
			"reply_count": row.ReplyCount,
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
			// 处理当前语句逻辑。
			"replies": replies,
		})
	}
	// 返回当前处理结果。
	return out, nil
}

// listForumCommentReplies 查询指定一级评论下的回复预览。
func (r *Repository) listForumCommentReplies(
	// 处理当前语句逻辑。
	ctx context.Context,
	// 处理当前语句逻辑。
	postID uint,
	// 处理当前语句逻辑。
	parentID uint,
	// 处理当前语句逻辑。
	limit int,
	// 进入新的代码块进行处理。
) ([]map[string]interface{}, error) {
	// 1) 参数保护。
	if parentID == 0 || limit <= 0 {
		// 返回当前处理结果。
		return []map[string]interface{}{}, nil
	}
	// 判断条件并进入对应分支逻辑。
	if limit > 20 {
		// 更新当前变量或字段值。
		limit = 20
	}

	// 2) 查询回复。
	rows := make([]commentRow, 0)
	// 定义并初始化当前变量。
	err := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_comment AS c").
		// 调用Select完成当前处理。
		Select(`c.id, c.user_id, c.parent_id, c.content, c.likes, c.created_at,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		// 更新当前变量或字段值。
		Joins("LEFT JOIN tk_users AS u ON u.id = c.user_id").
		// 更新当前变量或字段值。
		Where("c.status = 1 AND c.post_id = ? AND c.parent_id = ?", postID, parentID).
		// 调用Order完成当前处理。
		Order("c.created_at ASC, c.id ASC").
		// 调用Limit完成当前处理。
		Limit(limit).
		// 调用Scan完成当前处理。
		Scan(&rows).Error
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 3) 转换响应结构。
	out := make([]map[string]interface{}, 0, len(rows))
	// 循环处理当前数据集合。
	for _, row := range rows {
		// 更新当前变量或字段值。
		out = append(out, map[string]interface{}{
			// 处理当前语句逻辑。
			"id": row.ID,
			// 处理当前语句逻辑。
			"user_id": row.UserID,
			// 处理当前语句逻辑。
			"parent_id": row.ParentID,
			// 处理当前语句逻辑。
			"content": row.Content,
			// 处理当前语句逻辑。
			"likes": row.Likes,
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
	return out, nil
}

// buildForumTopicPayload 组装帖子主信息。
func buildForumTopicPayload(row topicRow) map[string]interface{} {
	// 1) 用户显示名兜底优先级：昵称 -> 用户名 -> 用户ID。
	displayName := strings.TrimSpace(row.Nickname)
	// 判断条件并进入对应分支逻辑。
	if displayName == "" {
		// 更新当前变量或字段值。
		displayName = strings.TrimSpace(row.Username)
	}
	// 判断条件并进入对应分支逻辑。
	if displayName == "" {
		// 更新当前变量或字段值。
		displayName = "用户"
	}

	// 2) 返回前端可直接消费的结构。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"id": row.ID,
		// 处理当前语句逻辑。
		"user_id": row.UserID,
		// 处理当前语句逻辑。
		"lottery_info_id": row.LotteryInfoID,
		// 处理当前语句逻辑。
		"title": row.Title,
		// 处理当前语句逻辑。
		"content": row.Content,
		// 处理当前语句逻辑。
		"cover_image": row.CoverImage,
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
			"display_name": displayName,
			// 处理当前语句逻辑。
			"avatar": row.Avatar,
			// 处理当前语句逻辑。
			"user_type": row.UserType,
		},
	}
}

// buildForumAuthorPayload 组装作者信息与统计字段。
func buildForumAuthorPayload(row topicRow, stats map[string]interface{}) map[string]interface{} {
	// 1) 作者显示名兜底。
	displayName := strings.TrimSpace(row.Nickname)
	// 判断条件并进入对应分支逻辑。
	if displayName == "" {
		// 更新当前变量或字段值。
		displayName = strings.TrimSpace(row.Username)
	}
	// 判断条件并进入对应分支逻辑。
	if displayName == "" {
		// 更新当前变量或字段值。
		displayName = "用户"
	}

	// 2) 输出作者结构。
	return map[string]interface{}{
		// 处理当前语句逻辑。
		"id": row.UserID,
		// 处理当前语句逻辑。
		"username": row.Username,
		// 处理当前语句逻辑。
		"nickname": row.Nickname,
		// 处理当前语句逻辑。
		"display_name": displayName,
		// 处理当前语句逻辑。
		"avatar": row.Avatar,
		// 处理当前语句逻辑。
		"user_type": row.UserType,
		// 处理当前语句逻辑。
		"stats": stats,
	}
}
