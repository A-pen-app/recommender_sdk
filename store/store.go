package store

import (
	"context"
)

type RecommendStore interface {
	NotifyStickiness(ctx context.Context, userID, postID string) error
}
