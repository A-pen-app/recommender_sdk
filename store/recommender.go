package store

import (
	"context"
	"sort"

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
const femaleFactor float64 = 4.

func (r *recommendStore[T]) Recommend(ctx context.Context, candidates []T, userID string) {
	var weights map[string]float64 = make(map[string]float64)

	blacklistIDs, err := r.GetBlacklistedUserIDs(ctx)
	if err != nil {
		logging.Errorw(ctx, "failed to get user blacklist", "err", err)
	}
	blacklistMap := make(map[string]struct{})
	for _, id := range blacklistIDs {
		blacklistMap[id] = struct{}{}
	}

	// boost non-annonymous posts whose authors are not in blacklist
	for _, t := range candidates {
		id := t.GetID()
		if _, exist := blacklistMap[id]; !exist {
			if !t.GetIsAnonymous() {
				*t.GetWeight() = max(nonAnnonymousFactor, weights[id]*nonAnnonymousFactor)
			}
			if t.GetGender() == "Female" {
				*t.GetWeight() = max(femaleFactor, weights[id]*femaleFactor)
			}
		}
	}

	sort.Sort(model.Rankables[T](candidates))
}

func (r *recommendStore[T]) GetBlacklistedUserIDs(ctx context.Context) ([]string, error) {
	var userIDs []string

	if err := r.db.Select(
		&userIDs,
		`
		SELECT
			user_id
		FROM
			feed_user_blacklist
		`,
	); err != nil {
		return nil, err
	}

	return userIDs, nil
}
