package ai

import (
	"brevitas/db"
	"time"

	"github.com/pocketbase/pocketbase/models"
)

// remember to start taking into account output cost
const SIM_RESPONSE_TIME = 20
const SIM_COST_PER_1K_INPUT = 0.0015
const SIM_INPUT_COST = SIM_COST_PER_1K_INPUT * 4

const SIM_COST_PER_1K_OUTPUT = 0.002
const SIM_OUTPUT_COST = 1 * SIM_COST_PER_1K_OUTPUT

// need to set a fake max batch for this function for now, since we'll be batching requests to chatGPT
func GetPostRating(post *models.Record, interest *models.Record) (db.PostRating, error) {

	return db.PostRating{}, nil
}

func GetPostRatings(posts []*models.Record, interests []*models.Record) []db.PostRating {
	var ratings []db.PostRating

	for _, post := range posts {
		for _, interest := range interests {
			rating, err := GetPostRating(post, interest)
			if err != nil {

			}
			ratings = append(ratings, rating)
		}
	}

	time.Sleep(SIM_RESPONSE_TIME * time.Second)

	return ratings
}
