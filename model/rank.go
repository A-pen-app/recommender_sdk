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

// Sort orders the candidates by descending score under opts. Every item is
// scored once against a single reference time and the sort is stable, so
// equal scores keep their input order instead of the previous
// sort.Sort-with-live-time.Now() behaviour, where two comparisons of the same
// pair could disagree.
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

// Score is the hot-feed formula:
//
//	(upvote/UpvoteDivisor + comments + favorites + shares + watchDays)
//	  / (ageHours + 2)^2  × weight
//
// where weight is whatever GetWeight() holds (0 means unweighted).
func Score(p Rankable, opts Options) float64 {
	now := opts.Now().Unix()
	totalWatchDays := p.GetTotalWatchSeconds() / 86400
	engagement := float64(p.GetUpvote()/opts.UpvoteDivisor + p.GetCommentsCount() + p.GetFavoriteCount() + p.GetShareCount() + totalWatchDays)
	score := engagement / math.Pow(float64(now-p.GetCreatedAt())/3600+2, 2)
	if w := p.GetWeight(); w != nil && *w != 0. {
		score *= *w
	}
	return score
}
