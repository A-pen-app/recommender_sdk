package recommender

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/A-pen-app/logging"
	"github.com/A-pen-app/recommender_sdk/model"
	"github.com/A-pen-app/recommender_sdk/store"
)

const recommenderURL string = "https://recommender-490242039522.asia-east1.run.app/recommendations/%s"

type Recommender[T model.Rankable] struct {
	weightCh <-chan map[string]float64
	timeout  time.Duration
}

func NewRecommender[T model.Rankable](ctx context.Context, userID string) *Recommender[T] {
	ch := make(chan map[string]float64, 1)
	recommender := &Recommender[T]{
		weightCh: ch,
		timeout:  time.Second * 2,
	}
	go func() {
		if w, err := getWeights(ctx, userID); err != nil {
			logging.Infow(ctx, "failed getting weights", "err", err)
		} else {
			ch <- w
		}
	}()
	return recommender
}

func (r *Recommender[T]) Recommend(ctx context.Context, candidates []T) []T {
	var weights map[string]float64
	select {
	case weights = <-r.weightCh:
	case <-time.After(r.timeout):
		logging.Debug(ctx, "timeout for getting weights")
	}

	if weights != nil { // there is a map and the map is not nil
		logging.Infow(ctx, "assigning weights...")
		for i, t := range candidates {
			if v, exists := weights[t.GetID()]; exists {
				candidates[i].AssignWeight(v)
			}
		}
	} else {
		logging.Debug(ctx, "unable to get weights")
	}
	sort.Sort(model.Rankables[T](candidates))
	return candidates
}

func buildRecommenderURL(userID string) string {
	url := fmt.Sprintf(recommenderURL, userID)
	return url
}

func getWeights(ctx context.Context, userID string) (map[string]float64, error) {
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

	if stickiness := store.GetStickinessScore(ctx, userID); stickiness != nil {
		logging.Infow(ctx, "stickiness cache retrieved", "score_length", len(stickiness.Scores))
		logging.Debug(ctx, fmt.Sprintf("stickiness %+v", stickiness.Scores))
		for k, v := range stickiness.Scores {
			scores[k] = scores[k] * v
		}
	}

	return scores, nil
}
