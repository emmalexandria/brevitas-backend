package db

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

// feed cache time in seconds
// this is a testing value
const FeedCacheTime = 3600

//const FeedCacheTime = 3600

func RefreshFeed(app *pocketbase.PocketBase, c echo.Context, parser *gofeed.Parser, feedID string) error {
	feed := Feed{}
	err := app.Dao().DB().
		Select("name", "url").
		From("feeds").
		Where(dbx.NewExp("id = {:id}", dbx.Params{"id": feedID})).
		One(&feed)

	if err != nil {
		return c.JSON(http.StatusNotFound,
			struct {
				Message string
			}{Message: "Resource not found"})
	}

	//if its time to refresh the posts

	postCollection, err := app.Dao().FindCollectionByNameOrId("posts")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "")
	}

	parsedFeed, err := parser.ParseURL(feed.Url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError,
			struct {
				Message string
			}{Message: "Could not parse posts"})
	}

	posts := make([]Post, len(parsedFeed.Items))

	for i, item := range parsedFeed.Items {
		posts[i] = NewPostFromItem(item)
		posts[i].Feed = feedID
	}

	for _, p := range FilterNewPosts(app, feedID, posts) {
		record := models.NewRecord(postCollection)

		record.Set("title", p.Title)
		record.Set("description", p.Description)
		record.Set("url", p.Url)
		record.Set("published", p.Published)
		record.Set("feed", p.Feed)

		app.Dao().SaveRecord(record)
	}

	return nil
}

func GetFeedPosts(app *pocketbase.PocketBase) ([]Post, int) {
	dbPosts := []Post{}

	err := app.Dao().
		DB().
		Select("title", "description", "url", "feed", "published").
		From("posts").
		Limit(-1).
		All(&dbPosts)

	if err != nil {
		return nil, http.StatusNotFound
	}

	return dbPosts, http.StatusOK
}
