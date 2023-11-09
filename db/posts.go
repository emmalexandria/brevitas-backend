package db

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

// how long posts should be retained in DB for in seconds (currently 30 days)
const RetainPostTime = 2592000

func FilterNewPosts(app *pocketbase.PocketBase, feedID string, posts []Post) []Post {
	currPostUrls := []struct {
		Url string `db:"url"`
	}{}

	err := app.Dao().
		DB().
		Select("url").
		From("posts").
		Where(dbx.NewExp("feed = {:id}", dbx.Params{"id": feedID})).
		All(&currPostUrls)

	if err != nil {
		println("Failed to retrieve posts!")
		return nil
	}

	filteredPosts := []Post{}

	for _, post := range posts {
		valid := true
		postPublished, err := time.Parse(time.RFC3339, post.Published)
		if err != nil {
			println(fmt.Sprintf("Failed to parse time value on post %s", post.Title))
		}

		if postPublished.Before(time.Unix(time.Now().Unix()-RetainPostTime, 0).UTC()) {
			valid = false
		}
		for _, currPost := range currPostUrls {
			if post.Url == currPost.Url {
				valid = false
			}
		}
		if valid {
			filteredPosts = append(filteredPosts, post)
		}
	}

	return filteredPosts
}

func DeleteOldPosts(app *pocketbase.PocketBase) {
	earliestDate := time.Unix(time.Now().Unix()-RetainPostTime, 0).UTC()

	records, err := app.Dao().FindRecordsByFilter(
		"posts",                          // collection
		"published < {:date}",            // filter
		"",                               // sort
		0,                                // limit
		0,                                // offset
		dbx.Params{"date": earliestDate}, // optional filter params
	)

	if err != nil {
		println(err)
		return
	}

	for _, record := range records {
		err = app.Dao().DeleteRecord(record)
		if err != nil {
			fmt.Println(err)
		}
	}

}
