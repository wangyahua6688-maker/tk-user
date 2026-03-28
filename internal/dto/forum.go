package dto

// ForumTopicQuery 论坛列表查询参数。
type ForumTopicQuery struct {
	Limit   int
	Feed    string
	Keyword string
	Issue   string
	Year    int
}

// ForumHistoryFilters 论坛历史贴筛选结构。
type ForumHistoryFilters struct {
	Years        []int    `json:"years"`
	Issues       []string `json:"issues"`
	CurrentYear  int      `json:"current_year"`
	CurrentIssue string   `json:"current_issue"`
}

// ForumTopicListResult 论坛列表聚合结果。
type ForumTopicListResult struct {
	Items          []map[string]interface{} `json:"items"`
	Total          int                      `json:"total"`
	HistoryFilters ForumHistoryFilters      `json:"history_filters"`
}

// LotteryCommentGroups 为彩种详情页提供四组评论数据。
type LotteryCommentGroups struct {
	SystemComments []map[string]interface{} `json:"system_comments"`
	UserComments   []map[string]interface{} `json:"user_comments"`
	HotComments    []map[string]interface{} `json:"hot_comments"`
	LatestComments []map[string]interface{} `json:"latest_comments"`
}
