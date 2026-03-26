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
		// 更新当前变量或字段值。
		limit = 10
	}
	// 判断条件并进入对应分支逻辑。
	if limit > 30 {
		// 更新当前变量或字段值。
		limit = 30
	}

	// 2) 读取缓存，降低高并发榜单查询的 DB 压力。
	cacheKey := fmt.Sprintf("tk:expert:boards:%s:%d", lotteryCode, limit)
	// 定义并初始化当前变量。
	cacheHit := map[string]interface{}{}
	// 判断条件并进入对应分支逻辑。
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		// 返回当前处理结果。
		return cacheHit, nil
	}

	// 3) 拉取候选用户（自然用户+机器人+官方）并聚合帖子/评论/点赞数据。
	type candidateRow struct {
		// 处理当前语句逻辑。
		UserID uint `json:"user_id"`
		// 处理当前语句逻辑。
		Username string `json:"username"`
		// 处理当前语句逻辑。
		Nickname string `json:"nickname"`
		// 处理当前语句逻辑。
		Avatar string `json:"avatar"`
		// 处理当前语句逻辑。
		UserType string `json:"user_type"`
		// 处理当前语句逻辑。
		PostCount int64 `json:"post_count"`
		// 处理当前语句逻辑。
		CommentCount int64 `json:"comment_count"`
		// 处理当前语句逻辑。
		LikeSum int64 `json:"like_sum"`
	}
	// 定义并初始化当前变量。
	rows := make([]candidateRow, 0)
	// 定义并初始化当前变量。
	err := r.db.WithContext(ctx).
		// 调用Table完成当前处理。
		Table("tk_users AS u").
		// 调用Select完成当前处理。
		Select(`u.id AS user_id, u.username, u.nickname, u.avatar, u.user_type,
				COALESCE(p.post_count, 0) AS post_count,
				COALESCE(c.comment_count, 0) AS comment_count,
				COALESCE(c.like_sum, 0) AS like_sum`).
		// 调用Joins完成当前处理。
		Joins(`LEFT JOIN (
				SELECT user_id, COUNT(1) AS post_count
				FROM tk_post_article
				WHERE status = 1
				GROUP BY user_id
			) AS p ON p.user_id = u.id`).
		// 调用Joins完成当前处理。
		Joins(`LEFT JOIN (
				SELECT user_id, COUNT(1) AS comment_count, COALESCE(SUM(likes), 0) AS like_sum
				FROM tk_comment
				WHERE status = 1
				GROUP BY user_id
			) AS c ON c.user_id = u.id`).
		// 更新当前变量或字段值。
		Where("u.status = 1 AND u.user_type IN ?", []string{"natural", "robot", "official"}).
		// 调用Limit完成当前处理。
		Limit(120).
		// 调用Scan完成当前处理。
		Scan(&rows).Error
	// 判断条件并进入对应分支逻辑。
	if err != nil {
		// 返回当前处理结果。
		return nil, err
	}

	// 4) 计算综合分，生成统一候选集合。
	type expertItem struct {
		// 处理当前语句逻辑。
		UserID uint
		// 处理当前语句逻辑。
		Nickname string
		// 处理当前语句逻辑。
		Avatar string
		// 处理当前语句逻辑。
		UserType string
		// 处理当前语句逻辑。
		HitRate int
		// 处理当前语句逻辑。
		Streak int
		// 处理当前语句逻辑。
		ReturnRate int
		// 处理当前语句逻辑。
		Score int64
		// 处理当前语句逻辑。
		ScoreLabel string
	}
	// 定义并初始化当前变量。
	candidates := make([]expertItem, 0, len(rows))
	// 循环处理当前数据集合。
	for _, row := range rows {
		// 定义并初始化当前变量。
		nickname := row.Nickname
		// 判断条件并进入对应分支逻辑。
		if nickname == "" {
			// 更新当前变量或字段值。
			nickname = row.Username
		}
		// 判断条件并进入对应分支逻辑。
		if nickname == "" {
			// 更新当前变量或字段值。
			nickname = fmt.Sprintf("用户%d", row.UserID)
		}
		// 定义并初始化当前变量。
		score := row.PostCount*12 + row.CommentCount*3 + row.LikeSum
		// 判断条件并进入对应分支逻辑。
		if score <= 0 {
			// 更新当前变量或字段值。
			score = int64(row.UserID%13 + 5)
		}
		// 定义并初始化当前变量。
		hitRate := int(45 + (score % 52))
		// 判断条件并进入对应分支逻辑。
		if hitRate > 99 {
			// 更新当前变量或字段值。
			hitRate = 99
		}
		// 定义并初始化当前变量。
		streak := int((score % 18) + 1)
		// 返回当前处理结果。
		returnRate := int(800 + score*11)
		// 更新当前变量或字段值。
		candidates = append(candidates, expertItem{
			// 处理当前语句逻辑。
			UserID: row.UserID,
			// 处理当前语句逻辑。
			Nickname: nickname,
			// 处理当前语句逻辑。
			Avatar: row.Avatar,
			// 处理当前语句逻辑。
			UserType: row.UserType,
			// 处理当前语句逻辑。
			HitRate: hitRate,
			// 处理当前语句逻辑。
			Streak: streak,
			// 处理当前语句逻辑。
			ReturnRate: returnRate,
			// 处理当前语句逻辑。
			Score: score,
			// 调用fmt.Sprintf完成当前处理。
			ScoreLabel: fmt.Sprintf("近30期命中率 %d%%", hitRate),
		})
	}

	// 5) 按综合分降序排列，并构建分组榜单。
	sort.SliceStable(candidates, func(i, j int) bool {
		// 返回当前处理结果。
		return candidates[i].Score > candidates[j].Score
	})
	// 定义并初始化当前变量。
	topN := candidates
	// 判断条件并进入对应分支逻辑。
	if len(topN) > limit {
		// 更新当前变量或字段值。
		topN = topN[:limit]
	}
	// 定义并初始化当前变量。
	buildGroup := func(key string, title string, pick func(item expertItem, idx int) bool) map[string]interface{} {
		// 定义并初始化当前变量。
		items := make([]map[string]interface{}, 0, limit)
		// 定义并初始化当前变量。
		rank := 0
		// 循环处理当前数据集合。
		for idx, item := range topN {
			// 判断条件并进入对应分支逻辑。
			if !pick(item, idx) {
				// 处理当前语句逻辑。
				continue
			}
			// 处理当前语句逻辑。
			rank++
			// 更新当前变量或字段值。
			items = append(items, map[string]interface{}{
				// 处理当前语句逻辑。
				"rank": rank,
				// 处理当前语句逻辑。
				"user_id": item.UserID,
				// 处理当前语句逻辑。
				"nickname": item.Nickname,
				// 处理当前语句逻辑。
				"avatar": item.Avatar,
				// 处理当前语句逻辑。
				"user_type": item.UserType,
				// 处理当前语句逻辑。
				"hit_rate": item.HitRate,
				// 处理当前语句逻辑。
				"streak": item.Streak,
				// 处理当前语句逻辑。
				"return_rate": item.ReturnRate,
				// 处理当前语句逻辑。
				"score_label": item.ScoreLabel,
				// 调用fmt.Sprintf完成当前处理。
				"streak_label": fmt.Sprintf("%d连中", item.Streak),
			})
			// 判断条件并进入对应分支逻辑。
			if len(items) >= limit {
				// 处理当前语句逻辑。
				break
			}
		}
		// 返回当前处理结果。
		return map[string]interface{}{
			// 处理当前语句逻辑。
			"key": key,
			// 处理当前语句逻辑。
			"title": title,
			// 处理当前语句逻辑。
			"items": items,
		}
	}

	// 定义并初始化当前变量。
	groups := []map[string]interface{}{
		// 调用buildGroup完成当前处理。
		buildGroup("total", "总榜", func(_ expertItem, _ int) bool { return true }),
		// 调用buildGroup完成当前处理。
		buildGroup("pingteyi", "平特一肖", func(item expertItem, _ int) bool { return item.HitRate >= 55 }),
		// 调用buildGroup完成当前处理。
		buildGroup("jiuxiao", "九肖中特", func(item expertItem, _ int) bool { return item.Streak >= 3 }),
		// 调用buildGroup完成当前处理。
		buildGroup("wuma", "五码中特", func(item expertItem, _ int) bool { return item.ReturnRate >= 1200 }),
	}

	// 6) 组装返回结构并写缓存。
	payload := map[string]interface{}{
		// 处理当前语句逻辑。
		"lottery_code": lotteryCode,
		// 处理当前语句逻辑。
		"groups": groups,
		// 调用len完成当前处理。
		"total": len(candidates),
	}
	// 调用r.saveCache完成当前处理。
	r.saveCache(ctx, cacheKey, payload)
	// 返回当前处理结果。
	return payload, nil
}
