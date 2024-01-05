package feeds

import (
	"brevitas/ai"

	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func ParseSourceIntoPosts(sourceURL string, dao *daos.Dao, parser *gofeed.Parser) error {
	dbSource, err := dao.FindFirstRecordByFilter("sources", "url={:sourceURL}", dbx.Params{"sourceURL": sourceURL})
	if err != nil {
		return err
	}
	postCollection, err := dao.FindCollectionByNameOrId("posts")
	if err != nil {
		return err
	}

	source, err := parser.ParseURL(sourceURL)
	if err != nil {
		return err
	}

	for _, item := range source.Items {
		if PostExists(dao, item.Link) {
			continue
		}
		record := models.NewRecord(postCollection)

		record.Set("title", item.Title)
		record.Set("description", item.Description)
		record.Set("url", item.Link)
		record.Set("published", item.PublishedParsed)
		record.Set("source", dbSource.Id)

		if item.Image != nil {
			record.Set("image", item.Image.URL)
		}

		err = dao.SaveRecord(record)
		if err != nil {
			return err
		}
	}

	return nil
}

func PostExists(dao *daos.Dao, postURL string) bool {
	record, _ := dao.FindFirstRecordByFilter("posts", "url={:url}", dbx.Params{"url": postURL})

	if record != nil {
		return true
	}
	return false
}

func CreatePostRatings(posts []*models.Record, interests []*models.Record, dao *daos.Dao) error {
	ratings := ai.GetPostRatings(posts, interests)

	for _, rating := range ratings {
		collection, err := dao.FindCollectionByNameOrId("articles")
		if err != nil {
			return err
		}

		record := models.NewRecord(collection)

		record.Set("post", rating.Post.Id)
		record.Set("interest", rating.Interest.Id)
		record.Set("rating", rating.Rating)
	}

	return nil
}
