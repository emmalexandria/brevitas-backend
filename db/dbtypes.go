package db

import (
	"time"

	"github.com/mmcdole/gofeed"
)

const FeedCacheTime = 3600

type Feed struct {
	Name string `db:"name" json:"name"`
	Url  string `db:"url" json:"url"`
}

type Post struct {
	Title       string    `db:"title" json:"title"`
	Description string    `db:"description" json:"description"`
	Url         string    `db:"url" json:"url"`
	Published   time.Time `db:"published" json:"published"`
	Feed        string    `db:"feed" json:"feed"`
}

func NewPostFromItem(item *gofeed.Item) *Post {
	p := new(Post)

	p.Title = item.Title
	p.Description = item.Description
	p.Published = *item.PublishedParsed
	p.Url = item.Link

	return p
}
