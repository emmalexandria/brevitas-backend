package db

import (
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func ParseSourceIntoPosts(sourceURL string, dao *daos.Dao, parser *gofeed.Parser) error {
	dbSource, err := dao.FindFirstRecordByFilter("sources", "url={:sourceURL}", dbx.Params{"sourceURL": sourceURL})
	postCollection, err := dao.FindCollectionByNameOrId("posts")
	if err != nil {
		return err
	}

	source, err := parser.ParseURL(sourceURL)
	if err != nil {
		return err
	}

	for _, item := range source.Items {
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
