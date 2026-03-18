package repo

import "time"

// topicRow 帖子列表查询结果行（包含聚合评论数）。
type topicRow struct {
	// 处理当前语句逻辑。
	ID uint `json:"id"`
	// 处理当前语句逻辑。
	UserID uint `json:"user_id"`
	// 处理当前语句逻辑。
	LotteryInfoID uint `json:"lottery_info_id"`
	// 处理当前语句逻辑。
	Title string `json:"title"`
	// 处理当前语句逻辑。
	Content string `json:"content"`
	// 处理当前语句逻辑。
	IsOfficial int8 `json:"is_official"`
	// 处理当前语句逻辑。
	CoverImage string `json:"cover_image"`
	// 处理当前语句逻辑。
	CommentCount int64 `json:"comment_count"`
	// 处理当前语句逻辑。
	LikeCount int64 `json:"like_count"`
	// 处理当前语句逻辑。
	CreatedAt time.Time `json:"created_at"`
	// 处理当前语句逻辑。
	Issue string `json:"issue"`
	// 处理当前语句逻辑。
	Year int `json:"year"`
	// 处理当前语句逻辑。
	SpecialLotteryID uint `json:"special_lottery_id"`
	// 处理当前语句逻辑。
	Username string `json:"username"`
	// 处理当前语句逻辑。
	Nickname string `json:"nickname"`
	// 处理当前语句逻辑。
	Avatar string `json:"avatar"`
	// 处理当前语句逻辑。
	UserType string `json:"user_type"`
}

// commentRow 评论查询结果行（评论 + 用户信息）。
type commentRow struct {
	// 处理当前语句逻辑。
	ID uint `json:"id"`
	// 处理当前语句逻辑。
	UserID uint `json:"user_id"`
	// 处理当前语句逻辑。
	ParentID uint `json:"parent_id"`
	// 处理当前语句逻辑。
	Content string `json:"content"`
	// 处理当前语句逻辑。
	Likes int64 `json:"likes"`
	// 处理当前语句逻辑。
	ReplyCount int64 `json:"reply_count"`
	// 处理当前语句逻辑。
	CreatedAt time.Time `json:"created_at"`
	// 处理当前语句逻辑。
	Username string `json:"username"`
	// 处理当前语句逻辑。
	Nickname string `json:"nickname"`
	// 处理当前语句逻辑。
	Avatar string `json:"avatar"`
	// 处理当前语句逻辑。
	UserType string `json:"user_type"`
}

// LotteryCommentGroups 为彩种详情页提供四组评论数据。
type LotteryCommentGroups struct {
	// 处理当前语句逻辑。
	SystemComments []map[string]interface{} `json:"system_comments"`
	// 处理当前语句逻辑。
	UserComments []map[string]interface{} `json:"user_comments"`
	// 处理当前语句逻辑。
	HotComments []map[string]interface{} `json:"hot_comments"`
	// 处理当前语句逻辑。
	LatestComments []map[string]interface{} `json:"latest_comments"`
}
