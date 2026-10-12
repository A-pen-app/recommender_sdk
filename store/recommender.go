package store

import (
	"context"

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

// Recommend writes each candidate's weight and sorts the slice in place by
// descending score. With no opts it behaves exactly as the fixed-constant
// version did; see model.Options for what can be tuned.
func (r *recommendStore[T]) Recommend(ctx context.Context, candidates []T, userID string, opts ...model.Option) {
	o := model.ApplyOptions(opts...)

	for _, t := range candidates {
		*t.GetWeight() = identityWeight(t, o)
	}

	model.Rankables[T](candidates).Sort(o)
}

// identityWeight reproduces the historical author-identity weighting: a
// female author gets FemaleFactor, otherwise a non-anonymous author gets
// NonAnonymousFactor, otherwise the item is unweighted (0 == ×1 in Score).
// The factors replace rather than stack, as before.
func identityWeight[T model.Rankable](t T, o model.Options) float64 {
	switch {
	case t.GetGender() == "Female":
		return o.FemaleFactor
	case !t.GetIsAnonymous():
		return o.NonAnonymousFactor
	default:
		return 0.
	}
}
