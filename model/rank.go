package model

import (
	"math"
	"time"
)

type Rankable interface {
	GetID() string
	GetUpvote() int
	GetTotalWatchSeconds() int
	GetCommentsCount() int
	GetShareCount() int
	GetFavoriteCount() int
	GetCreatedAt() int64
	GetWeight() *float64
	GetIsAnonymous() bool
	GetGender() string
}

type Rankables[T Rankable] []T

func (a Rankables[T]) Len() int { return len(a) }
func (a Rankables[T]) Less(i, j int) bool {
	return postScore(a[i]) > postScore(a[j])
}
func (a Rankables[T]) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func postScore(p Rankable) float64 {
	now := time.Now().Unix()
	totalWatchDays := p.GetTotalWatchSeconds() / 86400
	score := float64(p.GetUpvote()/2+p.GetCommentsCount()+p.GetFavoriteCount()+p.GetShareCount()+totalWatchDays) /
		math.Pow(float64(now-p.GetCreatedAt())/3600+2, 2)
	if w := p.GetWeight(); w != nil && *w != 0. {
		score *= *w
	}
	return score
}
