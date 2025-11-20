package store

import (
	"context"

	"github.com/A-pen-app/recommender_sdk/model"
)

type RecommendStore[T model.Rankable] interface {
	NotifyStickiness(ctx context.Context, userID, postID string) error
	Recommend(ctx context.Context, candidates []T)
}
