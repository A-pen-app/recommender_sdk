package recommender

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/A-pen-app/logging"
)

const recommenderURL string = "https://recommender-490242039522.asia-east1.run.app/recommendations/%s"
const RecommendStrength float64 = 32.

func buildRecommenderURL(userID string) string {
	url := fmt.Sprintf(recommenderURL, userID)
	return url
}

func GetPostsRecommendScores(ctx context.Context, userID string) (map[string]float64, error) {
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
	logging.Debug(ctx, fmt.Sprintf("scores %+v", scores))

	return scores, nil
}
