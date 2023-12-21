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
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
)

func main() {
	app := pocketbase.New()
	parser := gofeed.NewParser()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))

		e.Router.GET("/api/brevitas/sources/:sourceID/posts", func(c echo.Context) error {
			sourceID := c.PathParam("sourceID")

			source, err := app.Dao().FindRecordById("sources", sourceID)
			if err != nil {
				return err
			}

			db.ParseSourceIntoPosts(source.GetString("url"), app.Dao(), parser)

			posts, err := app.Dao().FindRecordsByFilter("posts", "source={:sourceID}", "", 0, 0, dbx.Params{"sourceID": sourceID})
			if err != nil {
				return err
			}

			return c.JSON(http.StatusOK, posts)
		})

		e.Router.POST("/api/brevitas/sources", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.NoContent(http.StatusNotFound)
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
				sourceRecord, _ := txDao.FindFirstRecordByFilter("sources", "url={:url}", dbx.Params{"url": data.URL})
				if sourceRecord == nil {

					source := db.Source{
						Name:        feed.Title,
						Url:         data.URL,
						BaseUrl:     feed.Link,
						Description: feed.Description,
						Type:        feed.FeedType,
					}

					sourceRecord, err = db.CreateSourceRecord(txDao, source)
				}

				err = db.CreateUserSourceRecord(txDao, db.UserSource{Name: data.Name, Publication: data.Publication}, authRecord.Id, sourceRecord.Id)

				if err != nil {
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

		e.Router.PATCH("/api/brevitas/user_sources/:userSourceID", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.NoContent(http.StatusNotFound)
			}

			data := apis.RequestInfo(c).Data

			err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
				name := data["name"]
				publication := data["publication"]

				userSourceID := c.PathParam("userSourceID")

				source, err := txDao.FindRecordById("user_sources", userSourceID)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, "Failed to find record")
				}

				source.Set("name", name)
				source.Set("publication", publication)

				if err := txDao.SaveRecord(source); err != nil {
					return c.JSON(http.StatusInternalServerError, "Error saving record")
				}

				return nil
			})

			if err != nil {
				return err
			}
			return c.NoContent(http.StatusOK)
		})

		e.Router.DELETE("/api/brevitas/interests/:interestID", func(c echo.Context) error {
			interestID := c.PathParam("interestID")

			record, err := app.Dao().FindRecordById("interests", interestID)
			if err != nil {
				return err
			}

			app.Dao().DeleteRecord(record)

			return c.NoContent(http.StatusOK)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
