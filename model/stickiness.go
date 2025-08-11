package model

import "time"

type StickinessRecommendation struct {
	Scores    map[string]float64 `json:"scores"`
	CreatedAt int64              `json:"created_at"`
}

func NewStickinessRecommendation() *StickinessRecommendation {
	return &StickinessRecommendation{
		Scores:    map[string]float64{},
		CreatedAt: time.Now().Unix(),
	}
}
