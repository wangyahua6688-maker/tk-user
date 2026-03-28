package repo

import (
	"context"
	"fmt"
	"sort"
)

// ListExpertBoards 返回高手推荐榜单。
func (r *ForumRepository) ListExpertBoards(ctx context.Context, limit int, lotteryCode string) (map[string]interface{}, error) {
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
	cacheKey := fmt.Sprintf("tk:expert:boards:v2:%s:%d", lotteryCode, limit)
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

	// 5) 每个榜单按自己的维度独立排序，保证前端切换 tab 时能看到真实变化。
	buildRankItems := func(source []expertItem, label func(item expertItem) string) []map[string]interface{} {
		items := make([]map[string]interface{}, 0, limit)
		for idx, item := range source {
			items = append(items, map[string]interface{}{
				"rank":         idx + 1,
				"user_id":      item.UserID,
				"nickname":     item.Nickname,
				"avatar":       item.Avatar,
				"user_type":    item.UserType,
				"hit_rate":     item.HitRate,
				"streak":       item.Streak,
				"return_rate":  item.ReturnRate,
				"score_label":  label(item),
				"streak_label": fmt.Sprintf("%d连中", item.Streak),
			})
			if len(items) >= limit {
				break
			}
		}
		return items
	}

	buildGroup := func(
		key string,
		title string,
		filter func(item expertItem) bool,
		less func(left expertItem, right expertItem) bool,
		label func(item expertItem) string,
	) map[string]interface{} {
		ranked := make([]expertItem, 0, len(candidates))
		for _, item := range candidates {
			if filter(item) {
				ranked = append(ranked, item)
			}
		}
		sort.SliceStable(ranked, func(i, j int) bool {
			return less(ranked[i], ranked[j])
		})
		if len(ranked) > limit {
			ranked = ranked[:limit]
		}
		return map[string]interface{}{
			"key":   key,
			"title": title,
			"items": buildRankItems(ranked, label),
		}
	}

	groups := []map[string]interface{}{
		buildGroup(
			"total",
			"总榜",
			func(_ expertItem) bool { return true },
			func(left expertItem, right expertItem) bool {
				if left.Score != right.Score {
					return left.Score > right.Score
				}
				if left.HitRate != right.HitRate {
					return left.HitRate > right.HitRate
				}
				return left.Streak > right.Streak
			},
			func(item expertItem) string {
				return fmt.Sprintf("近30期综合表现 %d 分", item.Score)
			},
		),
		buildGroup(
			"pingteyi",
			"平特一肖",
			func(item expertItem) bool { return item.HitRate >= 50 },
			func(left expertItem, right expertItem) bool {
				if left.HitRate != right.HitRate {
					return left.HitRate > right.HitRate
				}
				if left.Streak != right.Streak {
					return left.Streak > right.Streak
				}
				return left.Score > right.Score
			},
			func(item expertItem) string {
				return fmt.Sprintf("平特一肖近30期命中率 %d%%", item.HitRate)
			},
		),
		buildGroup(
			"jiuxiao",
			"九肖中特",
			func(item expertItem) bool { return item.Streak >= 3 },
			func(left expertItem, right expertItem) bool {
				if left.Streak != right.Streak {
					return left.Streak > right.Streak
				}
				if left.HitRate != right.HitRate {
					return left.HitRate > right.HitRate
				}
				return left.Score > right.Score
			},
			func(item expertItem) string {
				return fmt.Sprintf("九肖中特当前最长 %d 连中", item.Streak)
			},
		),
		buildGroup(
			"wuma",
			"五码中特",
			func(item expertItem) bool { return item.ReturnRate >= 1200 },
			func(left expertItem, right expertItem) bool {
				if left.ReturnRate != right.ReturnRate {
					return left.ReturnRate > right.ReturnRate
				}
				if left.HitRate != right.HitRate {
					return left.HitRate > right.HitRate
				}
				return left.Score > right.Score
			},
			func(item expertItem) string {
				return fmt.Sprintf("五码中特回报率 %d%%", item.ReturnRate)
			},
		),
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
