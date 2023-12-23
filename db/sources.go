package db

import (
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func CreateSourceRecord(dao *daos.Dao, source Source) (*models.Record, error) {
	sourceCollection, err := dao.FindCollectionByNameOrId("sources")
	if err != nil {
		return nil, err
	}

	record := models.NewRecord(sourceCollection)
	record.Set("name", source.Name)
	record.Set("url", source.Url)
	record.Set("description", source.Description)
	record.Set("type", source.Type)
	record.Set("base_url", source.BaseUrl)

	err = dao.SaveRecord(record)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func CreateUserSourceRecord(dao *daos.Dao, userSource UserSource, userID string, sourceID string) error {
	collection, err := dao.FindCollectionByNameOrId("user_sources")
	if err != nil {
		return err
	}

	record := models.NewRecord(collection)

	record.Set("name", userSource.Name)
	record.Set("publication", userSource.Publication)
	record.Set("user", userID)
	record.Set("source", sourceID)

	err = dao.SaveRecord(record)
	if err != nil {
		return err
	}

	return nil
}

func ParseSourceIfNeeded(source *models.Record, dao *daos.Dao, parser *gofeed.Parser, parseDelay int) error {
	lastParsed := source.GetDateTime("last_parsed")
	if lastParsed.IsZero() || lastParsed.Time().Add(time.Duration(parseDelay*int(time.Second))).Before(time.Now().UTC()) {
		err := ParseSourceIntoPosts(source.GetString("url"), dao, parser)
		if err != nil {
			return err
		}
		source.Set("last_parsed", time.Now().UTC())
		dao.SaveRecord(source)
	} else {
		return nil
	}
	return nil
}
