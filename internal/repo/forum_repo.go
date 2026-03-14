package repo

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ListForumTopics 返回论坛帖子列表（支持分栏、关键字、历史期号筛选）。
func (r *Repository) ListForumTopics(ctx context.Context, query ForumTopicQuery) (ForumTopicListResult, error) {
	// 1) 统一兜底 limit，避免一次性扫表过大。
	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 120 {
		query.Limit = 120
	}

	// 2) 归一化 feed/keyword 参数。
	feedKey := strings.ToLower(strings.TrimSpace(query.Feed))
	if feedKey == "" {
		feedKey = "all"
	}
	query.Feed = feedKey
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Issue = strings.TrimSpace(query.Issue)

	// 3) 历史贴模式先拉可选年份/期号，用于前端筛选器渲染与默认值计算。
	historyFilters := ForumHistoryFilters{}
	if feedKey == "history" {
		filters, err := r.loadForumHistoryFilters(ctx, query.Year, query.Issue)
		if err != nil {
			return ForumTopicListResult{}, err
		}
		historyFilters = filters
		// 历史贴默认锁定到当前年份，避免全量跨度过大。
		if query.Year <= 0 && filters.CurrentYear > 0 {
			query.Year = filters.CurrentYear
		}
	}

	// 4) 读取缓存，降低论坛高并发下的数据库压力。
	cacheKey := fmt.Sprintf(
		"tk:forum:topics:%s:%s:%s:%d:%d",
		query.Feed,
		strings.ToLower(query.Keyword),
		query.Issue,
		query.Year,
		query.Limit,
	)
	cacheHit := ForumTopicListResult{}
	if ok := r.loadCache(ctx, cacheKey, &cacheHit); ok {
		// 历史筛选器不走缓存，避免管理员新增期号后出现延迟。
		cacheHit.HistoryFilters = historyFilters
		return cacheHit, nil
	}

	// 5) 构建主查询，聚合评论数、点赞热度、用户信息、期号信息。
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
		Where("p.status = 1")

	// 6) 关键字筛选：标题优先，正文兜底。
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("p.title LIKE ? OR p.content LIKE ?", like, like)
	}

	// 7) 历史贴支持按年份/期号筛选。
	if feedKey == "history" {
		if query.Year > 0 {
			q = q.Where("li.year = ?", query.Year)
		}
		if query.Issue != "" {
			q = q.Where("li.issue = ?", query.Issue)
		}
	}

	// 8) 按分栏控制排序策略。
	switch feedKey {
	case "latest":
		q = q.Order("p.created_at DESC, p.id DESC")
	case "history":
		q = q.Order("li.year DESC, li.issue DESC, p.id DESC")
	default:
		// “全部”按点赞热度排序。
		q = q.Order("COALESCE(lk.like_count, 0) DESC, p.id DESC")
	}

	// 9) 执行查询。
	if err := q.Limit(query.Limit).Scan(&rows).Error; err != nil {
		return ForumTopicListResult{}, err
	}

	// 10) 转换为对外 JSON 结构。
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]interface{}{
			"id":                 row.ID,
			"user_id":            row.UserID,
			"lottery_info_id":    row.LotteryInfoID,
			"title":              row.Title,
			"is_official":        row.IsOfficial == 1,
			"cover_image":        row.CoverImage,
			"content_preview":    trimTextPreview(row.Content, 82),
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

	// 11) 组装最终结果。
	result := ForumTopicListResult{
		Items:          items,
		Total:          len(items),
		HistoryFilters: historyFilters,
	}
	// 12) 写缓存，保障后续同参数请求快速返回。
	r.saveCache(ctx, cacheKey, result)
	return result, nil
}

// loadForumHistoryFilters 加载历史贴筛选器（年份 + 期号）。
func (r *Repository) loadForumHistoryFilters(
	ctx context.Context,
	requestedYear int,
	requestedIssue string,
) (ForumHistoryFilters, error) {
	// 1) 查询所有“已发帖且关联图纸期号”的年份与期号。
	rows := make([]forumIssueRow, 0)
	err := r.db.WithContext(ctx).
		Table("tk_post_article AS p").
		Select("li.year, li.issue").
		Joins("INNER JOIN tk_lottery_info AS li ON li.id = p.lottery_info_id").
		Where("p.status = 1 AND li.status = 1").
		Where("li.year > 0 AND li.issue <> ''").
		Group("li.year, li.issue").
		Order("li.year DESC, li.issue DESC").
		Scan(&rows).Error
	if err != nil {
		return ForumHistoryFilters{}, err
	}

	// 2) 组装年份与期号映射。
	yearSet := make(map[int]struct{}, len(rows))
	issuesByYear := make(map[int][]string, len(rows))
	for _, row := range rows {
		yearSet[row.Year] = struct{}{}
		issuesByYear[row.Year] = append(issuesByYear[row.Year], row.Issue)
	}

	// 3) 年份去重后做倒序。
	years := make([]int, 0, len(yearSet))
	for y := range yearSet {
		years = append(years, y)
	}
	sort.SliceStable(years, func(i, j int) bool { return years[i] > years[j] })

	// 4) 计算当前选中年份：优先请求值，其次默认最新年份。
	currentYear := requestedYear
	if currentYear == 0 && len(years) > 0 {
		currentYear = years[0]
	}

	// 5) 读取当前年份下的期号并去重。
	issues := dedupeStrings(issuesByYear[currentYear])
	sort.SliceStable(issues, func(i, j int) bool { return issues[i] > issues[j] })

	// 6) 计算当前选中期号：请求值命中优先，否则取首项。
	currentIssue := strings.TrimSpace(requestedIssue)
	if currentIssue == "" && len(issues) > 0 {
		currentIssue = issues[0]
	}

	// 7) 返回筛选结构。
	return ForumHistoryFilters{
		Years:        years,
		Issues:       issues,
		CurrentYear:  currentYear,
		CurrentIssue: currentIssue,
	}, nil
}

// dedupeStrings 对字符串切片去重并保持原顺序。
func dedupeStrings(input []string) []string {
	// 1) 空切片直接返回。
	if len(input) == 0 {
		return []string{}
	}
	// 2) 逐个写入，已存在值跳过。
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// trimTextPreview 生成标题下方的正文预览文案。
func trimTextPreview(raw string, max int) string {
	// 1) 清理换行空白。
	value := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(raw, "\n", " "), "\r", " "))
	if max <= 0 || value == "" {
		return value
	}
	// 2) 不超长则直接返回。
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	// 3) 超长裁剪并追加省略号。
	return string(runes[:max]) + "..."
}
