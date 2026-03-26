package repo

import (
	"context"
	"fmt"
	"strings"
)

// LotteryCommentGroups 返回彩种详情页评论数据分组。
func (r *Repository) LotteryCommentGroups(ctx context.Context, infoID uint) (LotteryCommentGroups, error) {
	// 1) 评论分组优先查缓存。
	cacheKey := fmt.Sprintf("tk:comment:groups:%d", infoID)
	// 定义并初始化当前变量。
	cacheHit := LotteryCommentGroups{}
	// 判断条件并进入对应分支逻辑。
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		// 返回当前处理结果。
		return cacheHit, nil
	}

	// 2) 依次查询：系统评论、网友评论、热门评论、最新评论。
	systemRows, err := r.listLotteryComments(ctx, infoID, 12, "latest", []string{"official"})
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return LotteryCommentGroups{}, err
	}
	// 定义并初始化当前变量。
	userRows, err := r.listLotteryComments(ctx, infoID, 12, "latest", []string{"natural", "robot"})
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return LotteryCommentGroups{}, err
	}
	// 定义并初始化当前变量。
	hotRows, err := r.listLotteryComments(ctx, infoID, 8, "hot", nil)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return LotteryCommentGroups{}, err
	}
	// 定义并初始化当前变量。
	latestRows, err := r.listLotteryComments(ctx, infoID, 8, "latest", nil)
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return LotteryCommentGroups{}, err
	}

	// 3) 组装结构化结果。
	payload := LotteryCommentGroups{
		// 调用buildCommentPayload完成当前处理。
		SystemComments: buildCommentPayload(systemRows),
		// 调用buildCommentPayload完成当前处理。
		UserComments: buildCommentPayload(userRows),
		// 调用buildCommentPayload完成当前处理。
		HotComments: buildCommentPayload(hotRows),
		// 调用buildCommentPayload完成当前处理。
		LatestComments: buildCommentPayload(latestRows),
	}
	// 4) 缓存写回，供后续请求复用。
	r.saveCache(ctx, cacheKey, payload)
	// 返回当前处理结果。
	return payload, nil
}

// listLotteryComments 处理listLotteryComments相关逻辑。
func (r *Repository) listLotteryComments(ctx context.Context, infoID uint, limit int, orderBy string, userTypes []string) ([]commentRow, error) {
	// 1) 构建评论 + 用户信息联合查询。
	rows := make([]commentRow, 0)
	// 定义并初始化当前变量。
	q := r.db.WithContext(ctx).
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
		Where("c.status = 1 AND c.lottery_info_id = ?", infoID)
	// 判断条件并进入对应分支逻辑。
	if len(userTypes) > 0 {
		// 2) 仅筛选指定用户类型（如官方/机器人/自然用户）。
		q = q.Where("u.user_type IN ?", userTypes)
	}
	// 根据表达式进入多分支处理。
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "hot":
		// 3) 热门按点赞降序。
		q = q.Order("c.likes DESC, c.id DESC")
	default:
		// 4) 默认按发布时间降序。
		q = q.Order("c.created_at DESC, c.id DESC")
	}
	// 判断条件并进入对应分支逻辑。
	if limit > 0 {
		// 5) 控制单组最大返回条数。
		q = q.Limit(limit)
	}
	// 定义并初始化当前变量。
	err := q.Scan(&rows).Error
	// 返回当前处理结果。
	return rows, err
}

// buildCommentPayload 处理buildCommentPayload相关逻辑。
func buildCommentPayload(rows []commentRow) []map[string]interface{} {
	// 将数据库行映射为接口返回 JSON 结构。
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
			// 处理当前语句逻辑。
			"username": row.Username,
			// 处理当前语句逻辑。
			"nickname": row.Nickname,
			// 处理当前语句逻辑。
			"avatar": row.Avatar,
			// 处理当前语句逻辑。
			"user_type": row.UserType,
		})
	}
	// 返回当前处理结果。
	return out
}
