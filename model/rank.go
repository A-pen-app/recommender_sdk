package model

import (
	"math"
	"sort"
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

func (a Rankables[T]) Sort(opts Options) {
	scores := make([]float64, len(a))
	for i, t := range a {
		scores[i] = Score(t, opts)
	}
	idx := make([]int, len(a))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool { return scores[idx[i]] > scores[idx[j]] })
	sorted := make([]T, len(a))
	for i, k := range idx {
		sorted[i] = a[k]
	}
	copy(a, sorted)
}

func Score(p Rankable, opts Options) float64 {
	now := opts.Now().Unix()
	watchPoints := float64(p.GetTotalWatchSeconds() / opts.WatchSecondsDivisor)
	if opts.WatchScoreCap > 0 && watchPoints > opts.WatchScoreCap {
		watchPoints = opts.WatchScoreCap
	}
	engagement := float64(p.GetUpvote()/opts.UpvoteDivisor+p.GetCommentsCount()+p.GetFavoriteCount()+p.GetShareCount()) + watchPoints
	score := engagement / math.Pow(float64(now-p.GetCreatedAt())/3600+2, 2)
	if w := p.GetWeight(); w != nil && *w != 0. {
		score *= *w
	}
	return score
}
