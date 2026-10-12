package store

import (
	"context"
	"testing"
	"time"

	"github.com/A-pen-app/recommender_sdk/model"
)

type item struct {
	id        string
	upvote    int
	watch     int
	comments  int
	createdAt int64
	weight    float64
	anonymous bool
	gender    string
}

func (i *item) GetID() string             { return i.id }
func (i *item) GetUpvote() int            { return i.upvote }
func (i *item) GetTotalWatchSeconds() int { return i.watch }
func (i *item) GetCommentsCount() int     { return i.comments }
func (i *item) GetShareCount() int        { return 0 }
func (i *item) GetFavoriteCount() int     { return 0 }
func (i *item) GetCreatedAt() int64       { return i.createdAt }
func (i *item) GetWeight() *float64       { return &i.weight }
func (i *item) GetIsAnonymous() bool      { return i.anonymous }
func (i *item) GetGender() string         { return i.gender }

func ids(items []*item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.id
	}
	return out
}

func TestRecommendDefaultsMatchHistoricalWeights(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	items := []*item{
		{id: "anon", upvote: 10, createdAt: now.Unix() - 3600, anonymous: true},
		{id: "named", upvote: 10, createdAt: now.Unix() - 3600},
		{id: "female", upvote: 10, createdAt: now.Unix() - 3600, gender: "Female", anonymous: true},
	}
	s := &recommendStore[*item]{}
	s.Recommend(context.Background(), items, "u", model.WithNow(now))

	if items[0].id != "female" || items[1].id != "named" || items[2].id != "anon" {
		t.Fatalf("order = %v, want [female named anon]", ids(items))
	}
	if items[0].weight != 4 || items[1].weight != 2 || items[2].weight != 0 {
		t.Fatalf("weights = %v %v %v, want 4 2 0", items[0].weight, items[1].weight, items[2].weight)
	}
}

func TestRecommendUpvoteDivisorOption(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	one := &item{id: "one-upvote", upvote: 1, createdAt: now.Unix() - 3600, anonymous: true}
	zero := &item{id: "no-upvote", upvote: 0, createdAt: now.Unix() - 3600, anonymous: true}

	// Historical ÷2: a single upvote is worth nothing, so the order is the stable input order.
	s := &recommendStore[*item]{}
	s.Recommend(context.Background(), []*item{zero, one}, "u", model.WithNow(now))
	if model.Score(one, model.ApplyOptions(model.WithNow(now))) != 0 {
		t.Fatal("1 upvote / 2 should score 0 under defaults")
	}

	got := []*item{zero, one}
	s.Recommend(context.Background(), got, "u", model.WithNow(now), model.WithUpvoteDivisor(1))
	if got[0].id != "one-upvote" {
		t.Fatalf("order = %v, want one-upvote first with divisor 1", ids(got))
	}
}

func TestRecommendIsStableOnTies(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	items := []*item{}
	for _, id := range []string{"a", "b", "c", "d"} {
		items = append(items, &item{id: id, upvote: 4, createdAt: now.Unix() - 3600, anonymous: true})
	}
	s := &recommendStore[*item]{}
	s.Recommend(context.Background(), items, "u", model.WithNow(now))
	for i, want := range []string{"a", "b", "c", "d"} {
		if items[i].id != want {
			t.Fatalf("tie order changed: %v", ids(items))
		}
	}
}

func TestRecommendWatchDivisorAndCap(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	watched := &item{id: "watched", watch: 7200, createdAt: now.Unix() - 3600, anonymous: true}
	upvoted := &item{id: "upvoted", upvote: 2, createdAt: now.Unix() - 3600, anonymous: true}

	// Default ÷86400: two hours of watch time is worth nothing.
	if model.Score(watched, model.ApplyOptions(model.WithNow(now))) != 0 {
		t.Fatal("7200s / 86400 should score 0 under defaults")
	}

	got := []*item{upvoted, watched}
	s := &recommendStore[*item]{}
	s.Recommend(context.Background(), got, "u", model.WithNow(now), model.WithWatchSecondsDivisor(3600))
	if got[0].id != "watched" { // 2 watch points beat 2/2 = 1 upvote point
		t.Fatalf("order = %v, want watched first with divisor 3600", ids(got))
	}

	capped := model.Score(watched, model.ApplyOptions(model.WithNow(now), model.WithWatchSecondsDivisor(3600), model.WithWatchScoreCap(1)))
	uncapped := model.Score(watched, model.ApplyOptions(model.WithNow(now), model.WithWatchSecondsDivisor(3600)))
	if capped >= uncapped || capped == 0 {
		t.Fatalf("cap not applied: capped=%v uncapped=%v", capped, uncapped)
	}
}

func TestRecommendPersonalWeightsStackOnIdentity(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	named := &item{id: "named", upvote: 10, createdAt: now.Unix() - 3600}
	anon := &item{id: "anon", upvote: 10, createdAt: now.Unix() - 3600, anonymous: true}
	s := &recommendStore[*item]{}
	s.Recommend(context.Background(), []*item{named, anon}, "u", model.WithNow(now),
		model.WithWeights(map[string]float64{"named": 0.5, "anon": 1.3, "missing": 9}))
	if named.weight != 1 { // 2 × 0.5
		t.Fatalf("named weight = %v, want 1", named.weight)
	}
	if anon.weight != 1.3 { // unweighted identity (0) × 1.3 → 1.3, not 0
		t.Fatalf("anon weight = %v, want 1.3", anon.weight)
	}
}
