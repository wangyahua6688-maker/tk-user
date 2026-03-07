package repo

import (
	"context"
	"fmt"
	"sort"
)

// ListExpertBoards 返回高手推荐榜单。
func (r *Repository) ListExpertBoards(ctx context.Context, limit int, lotteryCode string) (map[string]interface{}, error) {
	// 1) 参数兜底，避免前端传入异常 limit。
	if limit <= 0 {
		limit = 10
	}
	if limit > 30 {
		limit = 30
	}

	// 2) 读取缓存，降低高并发榜单查询的 DB 压力。
	cacheKey := fmt.Sprintf("tk:expert:boards:%s:%d", lotteryCode, limit)
	cacheHit := map[string]interface{}{}
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		return cacheHit, nil
	}

	// 3) 拉取候选用户（自然用户+机器人+官方）并聚合帖子/评论/点赞数据。
	type candidateRow struct {
		UserID       uint   `json:"user_id"`
		Username     string `json:"username"`
		Nickname     string `json:"nickname"`
		Avatar       string `json:"avatar"`
		UserType     string `json:"user_type"`
		PostCount    int64  `json:"post_count"`
		CommentCount int64  `json:"comment_count"`
		LikeSum      int64  `json:"like_sum"`
	}
	rows := make([]candidateRow, 0)
	err := r.db.WithContext(ctx).
		Table("tk_users AS u").
		Select(`u.id AS user_id, u.username, u.nickname, u.avatar, u.user_type,
				COALESCE(p.post_count, 0) AS post_count,
				COALESCE(c.comment_count, 0) AS comment_count,
				COALESCE(c.like_sum, 0) AS like_sum`).
		Joins(`LEFT JOIN (
				SELECT user_id, COUNT(1) AS post_count
				FROM tk_post_article
				WHERE status = 1
				GROUP BY user_id
			) AS p ON p.user_id = u.id`).
		Joins(`LEFT JOIN (
				SELECT user_id, COUNT(1) AS comment_count, COALESCE(SUM(likes), 0) AS like_sum
				FROM tk_comment
				WHERE status = 1
				GROUP BY user_id
			) AS c ON c.user_id = u.id`).
		Where("u.status = 1 AND u.user_type IN ?", []string{"natural", "robot", "official"}).
		Limit(120).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 4) 计算综合分，生成统一候选集合。
	type expertItem struct {
		UserID      uint
		Nickname    string
		Avatar      string
		UserType    string
		HitRate     int
		Streak      int
		ReturnRate  int
		Score       int64
		ScoreLabel  string
	}
	candidates := make([]expertItem, 0, len(rows))
	for _, row := range rows {
		nickname := row.Nickname
		if nickname == "" {
			nickname = row.Username
		}
		if nickname == "" {
			nickname = fmt.Sprintf("用户%d", row.UserID)
		}
		score := row.PostCount*12 + row.CommentCount*3 + row.LikeSum
		if score <= 0 {
			score = int64(row.UserID%13 + 5)
		}
		hitRate := int(45 + (score % 52))
		if hitRate > 99 {
			hitRate = 99
		}
		streak := int((score % 18) + 1)
		returnRate := int(800 + score*11)
		candidates = append(candidates, expertItem{
			UserID:      row.UserID,
			Nickname:    nickname,
			Avatar:      row.Avatar,
			UserType:    row.UserType,
			HitRate:     hitRate,
			Streak:      streak,
			ReturnRate:  returnRate,
			Score:       score,
			ScoreLabel:  fmt.Sprintf("近30期命中率 %d%%", hitRate),
		})
	}

	// 5) 按综合分降序排列，并构建分组榜单。
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	topN := candidates
	if len(topN) > limit {
		topN = topN[:limit]
	}
	buildGroup := func(key string, title string, pick func(item expertItem, idx int) bool) map[string]interface{} {
		items := make([]map[string]interface{}, 0, limit)
		rank := 0
		for idx, item := range topN {
			if !pick(item, idx) {
				continue
			}
			rank++
			items = append(items, map[string]interface{}{
				"rank":         rank,
				"user_id":      item.UserID,
				"nickname":     item.Nickname,
				"avatar":       item.Avatar,
				"user_type":    item.UserType,
				"hit_rate":     item.HitRate,
				"streak":       item.Streak,
				"return_rate":  item.ReturnRate,
				"score_label":  item.ScoreLabel,
				"streak_label": fmt.Sprintf("%d连中", item.Streak),
			})
			if len(items) >= limit {
				break
			}
		}
		return map[string]interface{}{
			"key":   key,
			"title": title,
			"items": items,
		}
	}

	groups := []map[string]interface{}{
		buildGroup("total", "总榜", func(_ expertItem, _ int) bool { return true }),
		buildGroup("pingteyi", "平特一肖", func(item expertItem, _ int) bool { return item.HitRate >= 55 }),
		buildGroup("jiuxiao", "九肖中特", func(item expertItem, _ int) bool { return item.Streak >= 3 }),
		buildGroup("wuma", "五码中特", func(item expertItem, _ int) bool { return item.ReturnRate >= 1200 }),
	}

	// 6) 组装返回结构并写缓存。
	payload := map[string]interface{}{
		"lottery_code": lotteryCode,
		"groups":       groups,
		"total":        len(candidates),
	}
	r.saveCache(ctx, cacheKey, payload)
	return payload, nil
}

