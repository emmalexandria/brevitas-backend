package main

import (
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
				return c.JSON(http.StatusBadRequest, "Invalid url")
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
				feedRecord.Set("base_url", feed.FeedLink)

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

		e.Router.GET("/api/brevitas/feeds/:feedURL", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.JSON(http.StatusNotFound, "")
			}

			feed, err := parser.ParseURL(c.PathParam("feedURL"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, "Invalid url")
			}
			return c.JSON(http.StatusOK, map[string]any{"name": feed.Title, "description": feed.Description, "url": feed.FeedLink, "type": feed.FeedType, "image": feed.Image.URL})
		})

		e.Router.GET("/api/brevitas/feeds", func(c echo.Context) error {
			return c.JSON(http.StatusOK, true)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
