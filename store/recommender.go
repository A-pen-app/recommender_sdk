package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/A-pen-app/cache"
	"github.com/A-pen-app/logging"
	"github.com/A-pen-app/mq/v2"
	"github.com/A-pen-app/recommender_sdk/model"
	"github.com/jmoiron/sqlx"
)

func NewStore(q mq.MQ, db *sqlx.DB) *recommendStore {
	return &recommendStore{
		q:  q,
		db: db,
	}
}

type recommendStore struct {
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

func (r *recommendStore) NotifyStickiness(ctx context.Context, userID, postID string) error {
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
		if !t.GetIsAnnonymous() {
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

	if stickiness := GetStickinessScore(ctx, userID); stickiness != nil {
		logging.Infow(ctx, "stickiness cache retrieved", "score_length", len(stickiness.Scores))
		logging.Debug(ctx, fmt.Sprintf("stickiness %+v", stickiness.Scores))
		for k, v := range stickiness.Scores {
			scores[k] = scores[k] * v
		}
	}

	return scores, nil
}
