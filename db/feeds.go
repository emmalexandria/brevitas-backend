package db

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

func FilterNewPosts(app *pocketbase.PocketBase, feedID string, posts []Post) []Post {
	currPostUrls := []Post{}

	println(feedID)

	err := app.Dao().
		DB().
		Select("url").
		From("posts").
		Where(dbx.NewExp("feed = {:id}", dbx.Params{"id": feedID})).
		All(&currPostUrls)

	if err != nil {
		println(err)
	}
	filteredPosts := []Post{}

	println(len(currPostUrls))

	for _, post := range posts {
		valid := true
		for _, currPost := range currPostUrls {

			if post.Url == currPost.Url {
				valid = false
				break
			}
		}
		if valid {
			filteredPosts = append(filteredPosts, post)
		}
	}

	return filteredPosts
}
