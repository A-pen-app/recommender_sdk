package model

import (
	"math"
	"time"
)

type Rankable interface {
	GetID() string
	GetUpvote() int
	GetCommentsCount() int
	GetShareCount() int
	GetFavoriteCount() int
	GetCreatedAt() int64
	GetWeight() *float64
	GetIsAnnonymous() bool
}

type Rankables[T Rankable] []T

func (a Rankables[T]) Len() int { return len(a) }
func (a Rankables[T]) Less(i, j int) bool {
	return postScore(a[i]) > postScore(a[j])
}
func (a Rankables[T]) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func postScore(p Rankable) float64 {
	now := time.Now().Unix()
	score := float64(p.GetUpvote()/2+p.GetCommentsCount()+p.GetFavoriteCount()+p.GetShareCount()) /
		math.Pow(float64(now-p.GetCreatedAt())/3600+2, 1.8)
	if w := p.GetWeight(); w != nil && *w != 0. {
		score *= *w
	}
	return score
}
