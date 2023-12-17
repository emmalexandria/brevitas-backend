package main

import (
	"brevitas/backend/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

func main() {
	app := pocketbase.New()
	parser := gofeed.NewParser()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))

		e.Router.GET("/api/brevitas/feeds/:feedID", func(c echo.Context) error {

			return c.JSON(http.StatusOK, 200)
		})

		e.Router.POST("/api/brevitas/feeds", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.JSON(http.StatusNotFound, "")
			}

			data := struct {
				Name        string `json:"name" form:"name"`
				Publication string `json:"publication" form:"publication"`
				URL         string `json:"url" form:"url"`
			}{}

			if err := c.Bind(&data); err != nil {
				return apis.NewBadRequestError("Failed to read request data", err)
			}

			feed, err := parser.ParseURL(data.URL)
			if err != nil {
				fmt.Println(err)
				return c.JSON(http.StatusBadRequest, err)

			}

			err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				feedCollection, err := txDao.FindCollectionByNameOrId("feeds")
				if err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong fetching feeds")
				}

				feedRecord := models.NewRecord(feedCollection)
				feedRecord.Set("name", feed.Title)
				feedRecord.Set("url", data.URL)
				feedRecord.Set("description", feed.Description)
				feedRecord.Set("type", feed.FeedType)
				feedRecord.Set("base_url", feed.Link)

				if err := txDao.SaveRecord(feedRecord); err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong saving the publication")
				}

				feedUserCollection, err := txDao.FindCollectionByNameOrId("user_feeds")
				if err != nil {
					return err
				}

				feedUserRecord := models.NewRecord(feedUserCollection)
				feedUserRecord.Set("name", data.Name)
				feedUserRecord.Set("publication", data.Publication)
				feedUserRecord.Set("user", authRecord.Id)
				feedUserRecord.Set("feed", feedRecord.Id)

				if err := txDao.SaveRecord(feedUserRecord); err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong subscribing you to the publication")
				}

				return nil
			})

			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			return c.JSON(http.StatusOK, 200)
		})

		e.Router.GET("/api/brevitas/feeds/:feedID", func(c echo.Context) error {
			feedID := c.PathParam("feedID")

			feed, err := app.Dao().FindRecordById("feeds", feedID)
			if err != nil {
				return c.JSON(500, "Error fetching record")
			}

			parsedFeed, err := parser.ParseURL(feed.GetString("url"))
			if err != nil {
				return c.JSON(500, "Error parsing feed")
			}

			posts := parsedFeed.Items
			var ret_posts []db.Post

			for _, post := range posts {
				var post = db.Post{
					Title:       post.Title,
					Description: post.Description,
					Url:         post.Link,
					Published:   post.Published,
				}
				ret_posts = append(ret_posts, post)
			}

			json, err := json.Marshal(ret_posts)
			if err != nil {
				return c.JSON(500, "Error marshalling response")
			}

			return c.JSON(http.StatusOK, string(json))
		})

		e.Router.GET("/api/brevitas/feed", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.JSON(http.StatusNotFound, "")
			}

			return c.JSON(http.StatusOK, 200)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
