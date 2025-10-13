package store

import (
	"context"

	"github.com/A-pen-app/recommender_sdk/model"
)

type RecommendStore[T model.Rankable] interface {
	NotifyStickiness(ctx context.Context, userID, postID string) error
	NewRecommender(ctx context.Context, userID string) *Recommender[T]
}
