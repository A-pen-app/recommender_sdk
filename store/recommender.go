package store

import (
	"context"
	"sort"
	"time"

	"github.com/A-pen-app/cache"
	"github.com/A-pen-app/logging"
	"github.com/A-pen-app/mq/v2"
	"github.com/A-pen-app/recommender_sdk/model"
	"github.com/jmoiron/sqlx"
)

func NewStore[T model.Rankable](q mq.MQ, db *sqlx.DB) *recommendStore[T] {
	return &recommendStore[T]{
		q:  q,
		db: db,
	}
}

type recommendStore[T model.Rankable] struct {
	q  mq.MQ
	db *sqlx.DB
}

const stickiness string = "stickiness"

func GetStickinessScore(ctx context.Context, userID string) *model.StickinessRecommendation {
	r := model.NewStickinessRecommendation()
	if err := cache.Get(ctx, stickiness+":"+userID, r); err == nil {
		return r
	} else if err != cache.ErrorNotFound {
		// unexpected error, log down and skip
		logging.Errorw(ctx, "get user's recommend cache failed", "err", err, "user_id", userID)
	}
	return nil
}

func (r *recommendStore[T]) NotifyStickiness(ctx context.Context, userID, postID string) error {
	logging.Infow(ctx, "stickiness notified on user-post interaction event", "user_id", userID, "post_id", postID)

	if err := r.q.Send(stickiness, &model.RecommendEvent{
		UserID: userID,
		PostID: postID,
	}); err != nil {
		logging.Errorw(ctx, "send event failed", "err", err, "user_id", userID, "post_id", postID)
		return err
	}
	return nil
}

const nonAnnonymousFactor float64 = 2.

type Recommender[T model.Rankable] struct {
	weightCh <-chan map[string]float64
	timeout  time.Duration
}

func (r *recommendStore[T]) NewRecommender(ctx context.Context, userID string) *Recommender[T] {
	ch := make(chan map[string]float64, 1)
	recommender := &Recommender[T]{
		weightCh: ch,
		timeout:  time.Second * 2,
	}
	go func() {
		ch <- make(map[string]float64)
	}()
	return recommender
}

func (r *Recommender[T]) Recommend(ctx context.Context, candidates []T) {
	var weights map[string]float64

	select {
	case weights = <-r.weightCh:
	case <-time.After(r.timeout):
		logging.Debug(ctx, "timeout for getting weights")
	}

	if weights != nil { // there is a map and the map is not nil
		// weights["6c38770c-e187-40ec-8255-fffb66249a75"] = 10000
		logging.Debug(ctx, "assigning weights...")
		for _, t := range candidates {
			if w, exists := weights[t.GetID()]; exists {
				*t.GetWeight() = w
				// if *t.GetWeight() != 0 {
				// 	logging.Debug(ctx, fmt.Sprintf("[%f] assigned weight %f to %s", w, *t.GetWeight(), t.GetID()))
				// }
			}
		}
	}

	// add rule based approach
	// boost non-annonymous posts
	for _, t := range candidates {
		id := t.GetID()
		if !t.GetIsAnonymous() {
			// logging.Debug(ctx, fmt.Sprintf("[%s] is not annonymous", id))
			if w, exists := weights[id]; exists {
				weights[id] = w * nonAnnonymousFactor
			} else {
				*t.GetWeight() = nonAnnonymousFactor
			}
		}
	}

	sort.Sort(model.Rankables[T](candidates))
}
