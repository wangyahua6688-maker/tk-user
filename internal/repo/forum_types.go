package repo

import "tk-user/internal/dto"

// ForumTopicQuery 为论坛查询 DTO 的兼容别名。
type ForumTopicQuery = dto.ForumTopicQuery

// ForumHistoryFilters 为论坛筛选 DTO 的兼容别名。
type ForumHistoryFilters = dto.ForumHistoryFilters

// ForumTopicListResult 为论坛结果 DTO 的兼容别名。
type ForumTopicListResult = dto.ForumTopicListResult

// forumIssueRow 历史贴筛选期号行。
type forumIssueRow struct {
	// Year 年份。
	Year int `json:"year"`
	// Issue 期号。
	Issue string `json:"issue"`
}
