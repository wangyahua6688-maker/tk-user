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
	cacheHit := LotteryCommentGroups{}
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		return cacheHit, nil
	}

	// 2) 依次查询：系统评论、网友评论、热门评论、最新评论。
	systemRows, err := r.listLotteryComments(ctx, infoID, 12, "latest", []string{"official"})
	if err != nil {
		return LotteryCommentGroups{}, err
	}
	userRows, err := r.listLotteryComments(ctx, infoID, 12, "latest", []string{"natural", "robot"})
	if err != nil {
		return LotteryCommentGroups{}, err
	}
	hotRows, err := r.listLotteryComments(ctx, infoID, 8, "hot", nil)
	if err != nil {
		return LotteryCommentGroups{}, err
	}
	latestRows, err := r.listLotteryComments(ctx, infoID, 8, "latest", nil)
	if err != nil {
		return LotteryCommentGroups{}, err
	}

	// 3) 组装结构化结果。
	payload := LotteryCommentGroups{
		SystemComments: buildCommentPayload(systemRows),
		UserComments:   buildCommentPayload(userRows),
		HotComments:    buildCommentPayload(hotRows),
		LatestComments: buildCommentPayload(latestRows),
	}
	// 4) 缓存写回，供后续请求复用。
	r.saveCache(ctx, cacheKey, payload)
	return payload, nil
}

func (r *Repository) listLotteryComments(ctx context.Context, infoID uint, limit int, orderBy string, userTypes []string) ([]commentRow, error) {
	// 1) 构建评论 + 用户信息联合查询。
	rows := make([]commentRow, 0)
	q := r.db.WithContext(ctx).
		Table("tk_comment AS c").
		Select(`c.id, c.user_id, c.parent_id, c.content, c.likes, c.created_at,
				COALESCE(u.username, '') AS username,
				COALESCE(u.nickname, '') AS nickname,
				COALESCE(u.avatar, '') AS avatar,
				COALESCE(u.user_type, 'natural') AS user_type`).
		Joins("LEFT JOIN tk_users AS u ON u.id = c.user_id").
		Where("c.status = 1 AND c.lottery_info_id = ?", infoID)
	if len(userTypes) > 0 {
		// 2) 仅筛选指定用户类型（如官方/机器人/自然用户）。
		q = q.Where("u.user_type IN ?", userTypes)
	}
	switch strings.ToLower(strings.TrimSpace(orderBy)) {
	case "hot":
		// 3) 热门按点赞降序。
		q = q.Order("c.likes DESC, c.id DESC")
	default:
		// 4) 默认按发布时间降序。
		q = q.Order("c.created_at DESC, c.id DESC")
	}
	if limit > 0 {
		// 5) 控制单组最大返回条数。
		q = q.Limit(limit)
	}
	err := q.Scan(&rows).Error
	return rows, err
}

func buildCommentPayload(rows []commentRow) []map[string]interface{} {
	// 将数据库行映射为接口返回 JSON 结构。
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]interface{}{
			"id":         row.ID,
			"user_id":    row.UserID,
			"parent_id":  row.ParentID,
			"content":    row.Content,
			"likes":      row.Likes,
			"created_at": row.CreatedAt,
			"username":   row.Username,
			"nickname":   row.Nickname,
			"avatar":     row.Avatar,
			"user_type":  row.UserType,
		})
	}
	return out
}
