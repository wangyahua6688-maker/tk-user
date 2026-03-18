package repo

// ForumTopicQuery 论坛列表查询参数。
type ForumTopicQuery struct {
	// Limit 单次返回条数上限。
	Limit int
	// Feed 分栏键：all/latest/history。
	Feed string
	// Keyword 标题/正文搜索关键字。
	Keyword string
	// Issue 历史贴期号筛选（仅 history 生效）。
	Issue string
	// Year 历史贴年份筛选（仅 history 生效）。
	Year int
}

// ForumHistoryFilters 论坛历史贴筛选结构。
type ForumHistoryFilters struct {
	// Years 可选年份列表（倒序）。
	Years []int `json:"years"`
	// Issues 当前选中年份下的可选期号列表。
	Issues []string `json:"issues"`
	// CurrentYear 当前命中的年份。
	CurrentYear int `json:"current_year"`
	// CurrentIssue 当前命中的期号。
	CurrentIssue string `json:"current_issue"`
}

// ForumTopicListResult 论坛列表聚合结果。
type ForumTopicListResult struct {
	// Items 帖子列表。
	Items []map[string]interface{} `json:"items"`
	// Total 当前条件下的帖子数量。
	Total int `json:"total"`
	// HistoryFilters 历史贴筛选数据（仅 history feed 返回有效值）。
	HistoryFilters ForumHistoryFilters `json:"history_filters"`
}

// forumIssueRow 历史贴筛选期号行。
type forumIssueRow struct {
	// Year 年份。
	Year int `json:"year"`
	// Issue 期号。
	Issue string `json:"issue"`
}
