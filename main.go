package main

import (
	"log"
	"net/http"
	"os"

	"brevitas/backend/db"

	"github.com/labstack/echo/v5"
	"github.com/mmcdole/gofeed"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
)

func main() {
	app := pocketbase.New()
	parser := gofeed.NewParser()

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
		scheduler := cron.New()

		//run every day at 00:00
		scheduler.MustAdd("deleteOldPosts", "0 0 1-31 * *", func() {
			db.DeleteOldPosts(app)
			println("Old posts deleted")
		})

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

			//TODO: implement feed caching
			/* feedRecord, err := app.Dao().FindRecordById("feeds", c.PathParam("feedID"))

			if err != nil {
				return c.JSON(http.StatusNotFound,
					struct {
						Message string
					}{Message: "Feed not found"})
			}

			if int(time.Now().UTC().Sub(feedRecord.Updated.Time()).Seconds()) > db.FeedCacheTime {
				err = db.RefreshFeed(app, c, parser, c.PathParam("feedID"))
				if err != nil {
					return err
				}
			} */

			db.RefreshFeed(app, c, parser, c.PathParam("feedID"))
			dbPosts, code := db.GetFeedPosts(app)

			return c.JSON(code, dbPosts)

		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
