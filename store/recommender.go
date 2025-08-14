package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/A-pen-app/cache"
	"github.com/A-pen-app/logging"
	"github.com/A-pen-app/mq/v2"
	"github.com/A-pen-app/recommender_sdk/model"
)

func NewStore(q mq.MQ) *Store {
	return &Store{
		q: q,
	}
}

type Store struct {
	q mq.MQ
}

const stickiness string = "stickiness"
const recommenderURL string = "https://recommender-490242039522.asia-east1.run.app/recommendations/%s"

func buildRecommenderURL(userID string) string {
	url := fmt.Sprintf(recommenderURL, userID)
	return url
}

func (s *Store) GetPostsRecommendScores(ctx context.Context, userID string) (map[string]float64, error) {
	url := buildRecommenderURL(userID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logging.Infow(ctx, "failed creating request", "msg", err)
		return nil, err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.Infow(ctx, "error performing request", "msg", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		logging.Infow(ctx, "recommender respond with unexpected status code", "status_code", response.Status)
		return nil, err
	}

	var scores map[string]float64
	if err := json.NewDecoder(response.Body).Decode(&scores); err != nil {
		logging.Infow(ctx, "recommender error decoding score map", "msg", err)
		return nil, err
	}
	logging.Debug(ctx, fmt.Sprintf("similarity %+v", scores))

	if stickiness := getStickinessScore(ctx, userID); stickiness != nil {
		logging.Infow(ctx, "stickiness cache retrieved", "score_length", len(stickiness.Scores))
		logging.Debug(ctx, fmt.Sprintf("stickiness %+v", stickiness.Scores))
		for k, v := range stickiness.Scores {
			scores[k] = scores[k] * v
		}
	}

	return scores, nil
}

func getStickinessScore(ctx context.Context, userID string) *model.StickinessRecommendation {
	r := model.NewStickinessRecommendation()
	if err := cache.Get(ctx, stickiness+":"+userID, r); err == nil {
		return r
	} else if err != cache.ErrorNotFound {
		// unexpected error, log down and skip
		logging.Errorw(ctx, "get user's recommend cache failed", "err", err, "user_id", userID)
	}
	return nil
}

func (s *Store) NotifyStickiness(ctx context.Context, userID, postID string) error {
	if err := s.q.Send(stickiness, &model.RecommendEvent{
		UserID: userID,
		PostID: postID,
	}); err != nil {
		logging.Errorw(ctx, "send event failed", "err", err, "user_id", userID, "post_id", postID)
		return err
	}
	return nil
}
