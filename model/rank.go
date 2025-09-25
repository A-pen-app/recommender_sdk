package model

import (
	"math"
	"time"
)

type Rankable interface {
	GetID() string
	GetUpvote() int
	GetCommentsCount() int
	GetCreatedAt() int64
	GetRecommendWeight() float64
}

type Rankables[T Rankable] []T

func (a Rankables[T]) Len() int { return len(a) }
func (a Rankables[T]) Less(i, j int) bool {
	return postScore(a[i]) > postScore(a[j])
}
func (a Rankables[T]) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func postScore(p Rankable) float64 {
	now := time.Now().Unix()
	score := float64(p.GetUpvote()/2+p.GetCommentsCount()) /
		math.Pow(float64(now-p.GetCreatedAt())/3600+2, 1.5)
	if p.GetRecommendWeight() != 0 {
		score *= p.GetRecommendWeight()
	}
	return score
}
