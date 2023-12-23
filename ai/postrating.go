package ai

import "github.com/pocketbase/pocketbase/models"

// need to set a fake max batch for this function for now, since we'll be batching requests to chatGPT
func GetPostRating(post *models.Record, interest *models.Record) (int, error) {

	return 1, nil
}
