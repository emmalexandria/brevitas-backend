package db

import (
	"time"

	"github.com/mmcdole/gofeed"
)

type Feed struct {
	Name string `db:"name" json:"name"`
	Url  string `db:"url" json:"url"`
}

type Post struct {
	Title       string `db:"title" json:"title"`
	Description string `db:"description" json:"description"`
	Url         string `db:"url" json:"url"`
	Published   string `db:"published" json:"published"`
	Image       string `db:"image" json:"image"`
}

func NewPostFromItem(item *gofeed.Item) Post {
	p := Post{}

	p.Title = item.Title
	p.Description = item.Description
	p.Published = item.PublishedParsed.Format(time.RFC3339)
	p.Url = item.Link

	return p
}
