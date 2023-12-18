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

		e.Router.GET("/api/brevitas/sources/:sourceIDID", func(c echo.Context) error {

			return c.JSON(http.StatusOK, 200)
		})

		e.Router.POST("/api/brevitas/sources", func(c echo.Context) error {
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
				sourceCollection, err := txDao.FindCollectionByNameOrId("sources")
				if err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong fetching feeds")
				}

				sourceRecord := models.NewRecord(sourceCollection)
				sourceRecord.Set("name", feed.Title)
				sourceRecord.Set("url", data.URL)
				sourceRecord.Set("description", feed.Description)
				sourceRecord.Set("type", feed.FeedType)
				sourceRecord.Set("base_url", feed.Link)

				if err := txDao.SaveRecord(sourceRecord); err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong saving the publication")
				}

				userSourceCollection, err := txDao.FindCollectionByNameOrId("user_sources")
				if err != nil {
					return err
				}

				userSourceRecord := models.NewRecord(userSourceCollection)
				userSourceRecord.Set("name", data.Name)
				userSourceRecord.Set("publication", data.Publication)
				userSourceRecord.Set("user", authRecord.Id)
				userSourceRecord.Set("feed", sourceRecord.Id)

				if err := txDao.SaveRecord(userSourceRecord); err != nil {
					return c.String(http.StatusInternalServerError, "Something went wrong subscribing you to the publication")
				}

				return nil
			})

			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			return c.JSON(http.StatusOK, 200)
		})

		e.Router.GET("/api/brevitas/sources/:sourceID", func(c echo.Context) error {
			sourceID := c.PathParam("sourceID")

			source, err := app.Dao().FindRecordById("sources", sourceID)
			if err != nil {
				return c.JSON(500, "Error fetching record")
			}

			parsedSource, err := parser.ParseURL(source.GetString("url"))
			if err != nil {
				return c.JSON(500, "Error parsing feed")
			}

			posts := parsedSource.Items
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
