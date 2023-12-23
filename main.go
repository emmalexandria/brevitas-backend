package main

import (
	"brevitas/backend/db"
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
	"github.com/pocketbase/pocketbase/models"
)

// constant to use when determing when to parse a source
const SOURCE_PARSE_DELAY = 3600

func main() {
	app := pocketbase.New()
	parser := gofeed.NewParser()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))

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

		e.Router.GET("/api/brevitas/feed", func(c echo.Context) error {
			authRecord := apis.RequestInfo(c).AuthRecord
			if authRecord == nil {
				return c.NoContent(http.StatusNotFound)
			}
			userID := authRecord.Id

			//step 1: retrieve user_sources records
			userSources, err := app.Dao().FindRecordsByFilter("user_sources", "user.id={:userID}", "", 0, 0, dbx.Params{"userID": userID})

			var sources []*models.Record = make([]*models.Record, 0)
			for _, userSource := range userSources {
				sourceID := userSource.GetString("source")
				source, err := app.Dao().FindRecordById("sources", sourceID)

				if err != nil {
					return err
				}

				sources = append(sources, source)
				err = db.ParseSourceIfNeeded(source, app.Dao(), parser, SOURCE_PARSE_DELAY)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, err.Error())
				}
			}

			var posts []db.Post
			for _, source := range sources {
				postRecords, err := app.Dao().FindRecordsByFilter("posts", "source={:sourceID}", "", 0, 0, dbx.Params{"sourceID": source.Id})
				if err != nil {
					return err
				}
				var sourcePosts []db.Post

				for _, postRecord := range postRecords {
					userSource, err := app.Dao().FindFirstRecordByFilter("user_sources", "source={:sourceID}", dbx.Params{"sourceID": source.Id})
					if err != nil {
						return err
					}

					combSource := db.CombSource{
						Name:        userSource.GetString("name"),
						Publication: userSource.GetString("publication"),
						BaseUrl:     source.GetString("base_url"),
					}
					post := db.Post{
						Title:       postRecord.GetString("title"),
						Description: postRecord.GetString("description"),
						Url:         postRecord.GetString("url"),
						Published:   postRecord.GetDateTime("published"),
						Image:       postRecord.GetString("image"),
						Source:      combSource,
					}

					sourcePosts = append(sourcePosts, post)
				}

				posts = append(posts, sourcePosts...)

			}
			if err != nil {
				return err
			}

			return c.JSON(http.StatusOK, posts)
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
