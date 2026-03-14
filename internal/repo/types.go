package repo

import "time"

// topicRow 帖子列表查询结果行（包含聚合评论数）。
type topicRow struct {
	ID               uint      `json:"id"`
	UserID           uint      `json:"user_id"`
	LotteryInfoID    uint      `json:"lottery_info_id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	IsOfficial       int8      `json:"is_official"`
	CoverImage       string    `json:"cover_image"`
	CommentCount     int64     `json:"comment_count"`
	LikeCount        int64     `json:"like_count"`
	CreatedAt        time.Time `json:"created_at"`
	Issue            string    `json:"issue"`
	Year             int       `json:"year"`
	SpecialLotteryID uint      `json:"special_lottery_id"`
	Username         string    `json:"username"`
	Nickname         string    `json:"nickname"`
	Avatar           string    `json:"avatar"`
	UserType         string    `json:"user_type"`
}

// commentRow 评论查询结果行（评论 + 用户信息）。
type commentRow struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	ParentID   uint      `json:"parent_id"`
	Content    string    `json:"content"`
	Likes      int64     `json:"likes"`
	ReplyCount int64     `json:"reply_count"`
	CreatedAt  time.Time `json:"created_at"`
	Username   string    `json:"username"`
	Nickname   string    `json:"nickname"`
	Avatar     string    `json:"avatar"`
	UserType   string    `json:"user_type"`
}

// LotteryCommentGroups 为彩种详情页提供四组评论数据。
type LotteryCommentGroups struct {
	SystemComments []map[string]interface{} `json:"system_comments"`
	UserComments   []map[string]interface{} `json:"user_comments"`
	HotComments    []map[string]interface{} `json:"hot_comments"`
	LatestComments []map[string]interface{} `json:"latest_comments"`
}

// userAuthRow 用户认证信息查询结果。
type userAuthRow struct {
	ID           uint       `json:"id"`
	Username     string     `json:"username"`
	Phone        string     `json:"phone"`
	Nickname     string     `json:"nickname"`
	Avatar       string     `json:"avatar"`
	PasswordHash string     `json:"password_hash"`
	UserType     string     `json:"user_type"`
	Status       int8       `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}
