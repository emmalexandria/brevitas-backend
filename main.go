package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"brevitas/backend/db"

	"github.com/labstack/echo/v5"
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	app := pocketbase.New()
	parser := gofeed.NewParser()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))

		e.Router.GET("/api/brevitas/feeds/:feedID", func(c echo.Context) error {
			feed := db.Feed{}

			err := app.Dao().DB().
				Select("name", "url").
				From("feeds").
				Where(dbx.NewExp("id = {:id}", dbx.Params{"id": c.PathParam("feedID")})).
				One(&feed)

			if err != nil {
				return c.JSON(http.StatusNotFound,
					struct {
						Message string
					}{Message: "Resource not found"})
			}

			return c.JSON(http.StatusOK, feed)
		})

		e.Router.GET("/api/brevitas/feeds/:feedID/posts", func(c echo.Context) error {
			//check if the post list should be refreshed
			feedRecord, err := app.Dao().FindRecordById("feeds", c.PathParam("feedID"))

			if err != nil {
				return c.JSON(http.StatusNotFound,
					struct {
						Message string
					}{Message: "Feed not found"})
			}

			feed := db.Feed{}
			err = app.Dao().DB().
				Select("name", "url").
				From("feeds").
				Where(dbx.NewExp("id = {:id}", dbx.Params{"id": c.PathParam("feedID")})).
				One(&feed)

			if err != nil {
				return c.JSON(http.StatusNotFound,
					struct {
						Message string
					}{Message: "Resource not found"})
			}

			//if its time to refresh the posts
			if time.Now().Unix()-db.FeedCacheTime > feedRecord.Updated.Time().Unix() {
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

				posts := make([]db.Post, len(parsedFeed.Items))

				for i, item := range parsedFeed.Items {
					posts[i] = *db.NewPostFromItem(item)
					posts[i].Feed = c.PathParam("feedID")
				}

				for _, p := range db.FilterNewPosts(app, c.PathParam("feedID"), posts) {
					record := models.NewRecord(postCollection)

					record.Set("title", p.Title)
					record.Set("description", p.Description)
					record.Set("url", p.Url)
					record.Set("published", p.Published)
					record.Set("feed", p.Feed)

					//app.Dao().SaveRecord(record)
				}

			}

			dbPosts := []db.Post{}

			app.Dao().
				DB().
				Select("title", "description", "url", "published").
				From("posts").
				Where(dbx.NewExp("feed = {:id}", dbx.Params{"id": c.PathParam("feedID")})).
				All(dbPosts)

			return c.JSON(200, dbPosts)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
