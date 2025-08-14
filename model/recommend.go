package model

type RecommendEvent struct {
	UserID string `json:"user_id"`
	PostID string `json:"post_id"`
}
